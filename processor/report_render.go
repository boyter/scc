// SPDX-License-Identifier: MIT

package processor

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/mattn/go-isatty"
)

//go:embed report_template.gohtml
var reportTemplateSrc string

//go:embed report_card.gosvg
var reportCardSrc string

var (
	reportTmplOnce sync.Once
	reportTmpl     *template.Template
)

// reportTemplate parses the embedded HTML report and share-card templates on
// first use so a normal scc invocation pays nothing for the report path. The
// returned root template has both "report" and "card" defined.
func reportTemplate() *template.Template {
	reportTmplOnce.Do(func() {
		root := template.New("report").Funcs(reportFuncs)
		reportTmpl = template.Must(root.Parse(reportTemplateSrc))
		template.Must(reportTmpl.New("card").Parse(reportCardSrc))
	})
	return reportTmpl
}

// parseReportSkip turns the raw --report-skip value into the lower-cased
// lookup map CollectReportData consults via ReportSkipped. Empty input clears
// the map. Whitespace and case in each item are normalised. Unknown names
// emit a warning on stderr (per spec 05) and are still added to the map so
// that future template helpers can surface "you asked for X but it's not a
// real section" hints without re-parsing.
func parseReportSkip(raw string) {
	parseReportSkipTo(raw, os.Stderr)
}

// parseReportSkipTo is the testable seam for parseReportSkip — the warning
// destination is plumbed through so unit tests can capture stderr output
// without resorting to os.Stderr redirection.
func parseReportSkipTo(raw string, warnW io.Writer) {
	ReportSkipNames = map[string]bool{}
	if strings.TrimSpace(raw) == "" {
		return
	}
	for _, part := range strings.Split(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		if !reportSkipRecognised[name] {
			fmt.Fprintf(warnW, "warning: --report-skip: unknown section %q (recognised: cocomo, locomo, hotspots, authors, timeline, files, uloc, linelength, card)\n", name)
		}
		ReportSkipNames[name] = true
	}
}

// runReport is the dispatcher entry point invoked from Process() when
// ReportOut is set. It first prompts the user before clobbering an
// existing default-named file (so a bare `--report` is non-destructive),
// then collects data and renders the HTML output.
func runReport(paths []string) error {
	path := "."
	if len(paths) > 0 {
		path = paths[0]
	}
	usedDefault := ReportOut == DefaultReportName
	stdinIsTTY := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
	if err := confirmReportOverwrite(ReportOut, usedDefault, stdinIsTTY, os.Stdin, os.Stderr); err != nil {
		return err
	}
	data, err := CollectReportData(path)
	if err != nil {
		return err
	}
	return RenderReport(data, ReportOut)
}

// confirmReportOverwrite asks the user before clobbering a default-named
// scc-report.html in the current directory. The contract is:
//
//   - explicit path (`--report=foo.html`): treat the name as deliberate
//     consent and overwrite silently — usedDefaultName is false.
//   - bare `--report` and the file doesn't exist: proceed.
//   - bare `--report`, file exists, TTY attached: prompt "Overwrite? [y/N]"
//     and return nil only on an affirmative answer.
//   - bare `--report`, file exists, no TTY (CI, piped stdin): refuse and
//     point the user at the explicit-path form so the intent is auditable
//     in scripts.
//
// The io.Reader / io.Writer seam keeps the function unit-testable without
// poking at os.Stdin / os.Stderr.
func confirmReportOverwrite(outPath string, usedDefaultName, stdinIsTTY bool, in io.Reader, out io.Writer) error {
	if !usedDefaultName {
		return nil
	}
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		// Some other stat error (permission?) — let the subsequent
		// os.Create surface the underlying problem with a clearer
		// "create report file: ..." wrapper instead of a stat error
		// nobody asked for.
		return nil
	}
	if !stdinIsTTY {
		return fmt.Errorf("%s already exists; rerun with --report=%s to overwrite explicitly", outPath, outPath)
	}
	fmt.Fprintf(out, "%s already exists. Overwrite? [y/N]: ", outPath)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return fmt.Errorf("aborted: %w", err)
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	if ans == "y" || ans == "yes" {
		return nil
	}
	return fmt.Errorf("aborted: %s not overwritten", outPath)
}

// RenderReport renders the share card first (so the result can be embedded
// as og:image in the main template) and then writes the page to outPath.
func RenderReport(d ReportData, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	defer f.Close()

	if err := renderReportTo(f, d); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Report written to %s\n", outPath)
	return nil
}

// renderReportTo is the io.Writer-shaped seam used by the golden test in
// spec 06. It renders the share card (so the og:image data URL is
// available) and then writes the full HTML page in one pass. Callers that
// just want bytes can use a bytes.Buffer.
func renderReportTo(w io.Writer, d ReportData) error {
	if !ReportSkipped("card") {
		var cardBuf bytes.Buffer
		if err := reportTemplate().ExecuteTemplate(&cardBuf, "card", d); err != nil {
			return fmt.Errorf("render share card: %w", err)
		}
		d.CardSVG = template.HTML(cardBuf.String())
	}
	if err := reportTemplate().ExecuteTemplate(w, "report", d); err != nil {
		return fmt.Errorf("render report: %w", err)
	}
	return nil
}

// reportLangColors is the GitHub-style language colour palette consulted by
// the langColor template helper. Anything not in the map renders with the
// neutral fallback. Keep it short — exotic languages can fall back safely.
var reportLangColors = map[string]string{
	"Go":         "#00ADD8",
	"JavaScript": "#f1e05a",
	"TypeScript": "#3178c6",
	"Python":     "#3572A5",
	"Java":       "#b07219",
	"C":          "#555555",
	"C++":        "#f34b7d",
	"C#":         "#178600",
	"Ruby":       "#701516",
	"Rust":       "#dea584",
	"PHP":        "#4F5D95",
	"Swift":      "#F05138",
	"Kotlin":     "#A97BFF",
	"Shell":      "#89e051",
	"Bash":       "#89e051",
	"HTML":       "#e34c26",
	"CSS":        "#563d7c",
	"SCSS":       "#c6538c",
	"Markdown":   "#083fa1",
	"YAML":       "#cb171e",
	"JSON":       "#292929",
	"TOML":       "#9c4221",
	"XML":        "#0060ac",
	"SQL":        "#e38c00",
	"Dockerfile": "#384d54",
	"Makefile":   "#427819",
	"Perl":       "#0298c3",
	"Lua":        "#000080",
	"R":          "#198CE7",
	"Scala":      "#c22d40",
	"Haskell":    "#5e5086",
	"Elixir":     "#6e4a7e",
	"Erlang":     "#B83998",
	"Clojure":    "#db5855",
	"Vue":        "#41b883",
	"Svelte":     "#ff3e00",
	"Plain Text": "#999999",
	"Zig":        "#ec915c",
}

// DonutArc is the geometry for a single arc segment in a donut chart. Used
// by donutArcs and consumed via the dasharray/dashoffset SVG attributes on a
// <circle>. Spec calls for this even though the sample mockup uses a flat
// composition bar — included for future templates / share-card variants.
type DonutArc struct {
	Color      string
	Dasharray  string
	Dashoffset float64
}

// Bar is one bar in a histogram. Used by bucketBars for the line-length
// chart. X is the SVG x-coordinate, W and H are width/height in user units.
type Bar struct {
	X     int
	W     int
	H     int
	Y     int
	Count int64
	Label string
}

// reportFuncs is the template func map registered against both the page
// template and the share card. Everything in here is pure (no I/O, no
// globals beyond the colour map) so templates remain deterministic.
var reportFuncs = template.FuncMap{
	"comma": func(n int64) string {
		// Manual implementation avoids pulling text/message's MatchString
		// allocations for what is a hot template helper.
		if n < 0 {
			return "-" + commaFmt(-n)
		}
		return commaFmt(n)
	},
	"commaInt": func(n int) string {
		return commaFmt(int64(n))
	},
	"pct": func(num, denom int64) string {
		if denom == 0 {
			return "0.0%"
		}
		return fmt.Sprintf("%.1f%%", float64(num)/float64(denom)*100)
	},
	"pctFloat": func(v float64) string {
		return fmt.Sprintf("%.1f%%", v*100)
	},
	"pctRaw": func(num, denom int64) float64 {
		if denom == 0 {
			return 0
		}
		return float64(num) / float64(denom) * 100
	},
	"bytes":           humanBytes,
	"langColor":       langColor,
	"donutArcs":       donutArcs,
	"sparklinePath":   sparklinePath,
	"sparklineFill":   sparklineFill,
	"sparklinePath64": sparklinePath64,
	"sparklineFill64": sparklineFill64,
	"bucketBars":      bucketBars,
	"histoBars":       histoBars,
	"authorActivity":  authorActivity,
	"durationSeconds": func(d interface{}) float64 {
		switch v := d.(type) {
		case float64:
			return v
		default:
			// time.Duration formatted via Seconds() method.
			if ds, ok := d.(interface{ Seconds() float64 }); ok {
				return ds.Seconds()
			}
			_ = v
			return 0
		}
	},
	"divFloat": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"mulFloat":    func(a, b float64) float64 { return a * b },
	"intCast":     func(v float64) int64 { return int64(v) },
	"dataURLCard": dataURLCard,
	"safeHTML":    func(s string) template.HTML { return template.HTML(s) },
	"skipped":     func(section string) bool { return ReportSkipped(section) },
	"add":         func(a, b int) int { return a + b },
	"sub":         func(a, b int) int { return a - b },
	"mul":         func(a, b int) int { return a * b },
	"div": func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"int64":     func(n int) int64 { return int64(n) },
	"fromInt64": func(n int64) int { return int(n) },
	"fmtTime": func(layout string, t interface{}) string {
		switch v := t.(type) {
		case string:
			return v
		default:
			return fmt.Sprintf("%v", t)
		}
	},
	"firstN": func(n int, items interface{}) interface{} {
		// Generic slice truncation via reflection-free type switch on the
		// concrete slice types the template uses.
		switch s := items.(type) {
		case []LanguageSummary:
			if len(s) > n {
				return s[:n]
			}
			return s
		case []HotspotRow:
			if len(s) > n {
				return s[:n]
			}
			return s
		case []AuthorRow:
			if len(s) > n {
				return s[:n]
			}
			return s
		case []LangTimelineRow:
			if len(s) > n {
				return s[:n]
			}
			return s
		case []AuthorTimelineRow:
			if len(s) > n {
				return s[:n]
			}
			return s
		case []*FileJob:
			if len(s) > n {
				return s[:n]
			}
			return s
		case []LineLengthOutlier:
			if len(s) > n {
				return s[:n]
			}
			return s
		}
		return items
	},
	"sliceLen": func(items interface{}) int {
		switch s := items.(type) {
		case []LanguageSummary:
			return len(s)
		case []HotspotRow:
			return len(s)
		case []AuthorRow:
			return len(s)
		case []LangTimelineRow:
			return len(s)
		case []AuthorTimelineRow:
			return len(s)
		case []*FileJob:
			return len(s)
		case []LineLengthOutlier:
			return len(s)
		case []LineLengthBucket:
			return len(s)
		}
		return 0
	},
	"deltaSign": func(n int64) string {
		if n > 0 {
			return "+"
		}
		return ""
	},
	"deltaColor": func(n int64) string {
		if n > 0 {
			return "var(--good)"
		}
		if n < 0 {
			return "var(--danger)"
		}
		return "var(--fg-muted)"
	},
	// filesSorted returns the file list sorted by line count descending so the
	// "Notable files" table is deterministic regardless of walker order.
	"filesSorted": func(files []*FileJob) []*FileJob {
		out := make([]*FileJob, len(files))
		copy(out, files)
		sort.Slice(out, func(i, j int) bool {
			if out[i].Lines != out[j].Lines {
				return out[i].Lines > out[j].Lines
			}
			return out[i].Location < out[j].Location
		})
		return out
	},
}

func commaFmt(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		if len(s) > pre {
			b.WriteByte(',')
		}
	}
	for i := pre; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	return b.String()
}

// humanBytes renders a byte count as "612K", "4.2M", etc. Uses SI units
// (1000-based) to match the existing tabular formatter default. Output is
// kept compact for table cells and the share card.
func humanBytes(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"K", "M", "G", "T", "P"}
	if exp >= len(suffixes) {
		exp = len(suffixes) - 1
	}
	v := float64(n) / float64(div)
	if v >= 100 {
		return fmt.Sprintf("%.0f%s", v, suffixes[exp])
	}
	if v >= 10 {
		return fmt.Sprintf("%.1f%s", v, suffixes[exp])
	}
	return fmt.Sprintf("%.1f%s", v, suffixes[exp])
}

func langColor(name string) string {
	if c, ok := reportLangColors[name]; ok {
		return c
	}
	return "#999999"
}

// donutArcs converts a sorted-by-code LanguageSummary slice into the geometry
// the donut SVG needs. circumference 2πr with r=1 keeps arithmetic simple —
// callers scale via the SVG circle's stroke-dasharray with the canonical r.
func donutArcs(summary []LanguageSummary) []DonutArc {
	var total int64
	for _, s := range summary {
		total += s.Code
	}
	if total == 0 {
		return nil
	}
	const circ = 100.0
	offset := 0.0
	arcs := make([]DonutArc, 0, len(summary))
	for _, s := range summary {
		frac := float64(s.Code) / float64(total)
		seg := frac * circ
		arcs = append(arcs, DonutArc{
			Color:      langColor(s.Name),
			Dasharray:  fmt.Sprintf("%.3f %.3f", seg, circ-seg),
			Dashoffset: -offset,
		})
		offset += seg
	}
	return arcs
}

// sparklinePath turns a series into an SVG `d=` attribute scaled into a
// w×h box. Linear interpolation between data points. Empty input returns "".
func sparklinePath(values []int, w, h int) string {
	return sparklinePathInternal(intsTo64(values), w, h, false)
}

func sparklineFill(values []int, w, h int) string {
	return sparklinePathInternal(intsTo64(values), w, h, true)
}

func sparklinePath64(values []int64, w, h int) string {
	return sparklinePathInternal(values, w, h, false)
}

func sparklineFill64(values []int64, w, h int) string {
	return sparklinePathInternal(values, w, h, true)
}

func intsTo64(in []int) []int64 {
	out := make([]int64, len(in))
	for i, v := range in {
		out[i] = int64(v)
	}
	return out
}

func sparklinePathInternal(values []int64, w, h int, closed bool) string {
	if len(values) == 0 || w <= 0 || h <= 0 {
		return ""
	}
	minV, maxV := values[0], values[0]
	for _, v := range values {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	span := float64(maxV - minV)
	if span == 0 {
		span = 1
	}
	stepX := float64(w)
	if len(values) > 1 {
		stepX = float64(w) / float64(len(values)-1)
	}
	var b strings.Builder
	for i, v := range values {
		x := float64(i) * stepX
		// Invert Y so larger values are higher.
		y := float64(h) - (float64(v-minV)/span)*float64(h)
		if i == 0 {
			fmt.Fprintf(&b, "M%.1f,%.1f", x, y)
		} else {
			fmt.Fprintf(&b, " L%.1f,%.1f", x, y)
		}
	}
	if closed {
		// Close the path to the baseline so the area fill renders cleanly.
		fmt.Fprintf(&b, " L%.1f,%.1f L%.1f,%.1f Z", float64(w), float64(h), 0.0, float64(h))
	}
	return b.String()
}

// bucketBars converts a slice of bucket counts into Bar geometry sized into
// a maxH-tall plot. Bars are uniformly spaced; the caller positions the SVG
// at whatever x-origin it wants by adding to Bar.X.
func bucketBars(buckets []int64, maxH int) []Bar {
	if len(buckets) == 0 || maxH <= 0 {
		return nil
	}
	var max int64
	for _, b := range buckets {
		if b > max {
			max = b
		}
	}
	if max == 0 {
		max = 1
	}
	const barW = 80
	const gap = 10
	bars := make([]Bar, len(buckets))
	for i, count := range buckets {
		h := int(math.Round(float64(count) / float64(max) * float64(maxH)))
		if h < 1 && count > 0 {
			h = 1
		}
		bars[i] = Bar{
			X:     i*(barW+gap) + 10,
			W:     barW,
			H:     h,
			Y:     maxH - h,
			Count: count,
		}
	}
	return bars
}

// histoBars projects the line-length histogram buckets directly into Bar
// geometry. Saves the template from having to extract the Count field into
// a parallel slice. baseX shifts each bar's X origin so the caller can place
// the histogram beside an axis.
func histoBars(buckets []LineLengthBucket, plotH int) []Bar {
	counts := make([]int64, len(buckets))
	for i, b := range buckets {
		counts[i] = b.Count
	}
	bars := bucketBars(counts, plotH)
	for i := range bars {
		bars[i].Label = buckets[i].Label
		bars[i].Count = buckets[i].Count
	}
	return bars
}

// authorActivity flattens an author's per-bucket Series into a list of code
// deltas suitable for sparklinePath64. Used by the timeline template.
func authorActivity(series []AuthorTimelineBucket) []int64 {
	out := make([]int64, len(series))
	for i, b := range series {
		out[i] = b.CodeDelta
	}
	return out
}

// dataURLCard URL-encodes the rendered card SVG into a data: URL suitable
// for og:image. Uses a minimal escape set so the URL stays compact and
// human-readable when viewing source.
func dataURLCard(card template.HTML) string {
	s := string(card)
	// Strip leading whitespace / newlines so the resulting URL is compact.
	s = strings.TrimSpace(s)
	// QueryEscape encodes too aggressively for our purposes (e.g. spaces -> +)
	// but it's safe for og:image consumers, so we use it then patch the few
	// edge cases.
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	return "data:image/svg+xml;utf8," + encoded
}
