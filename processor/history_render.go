// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
)

// historyDateLayout is the canonical date-only format used in window
// headers and metadata. Matches the spec example "2024-01-09 → 2026-05-20".
const historyDateLayout = "2006-01-02"

// historyHeader renders the centred two-line "<break> <name> · last N
// commits · from → to <break>" block that every tabular report uses.
func historyHeader(reportName string, w HistoryWindow, wide bool) string {
	break_ := tabularBreakFor(wide)
	var sb strings.Builder
	sb.WriteString(break_)
	sb.WriteString(formatHeaderLine(reportName, w))
	sb.WriteByte('\n')
	sb.WriteString(break_)
	return sb.String()
}

func formatHeaderLine(reportName string, w HistoryWindow) string {
	if w.Commits == 0 {
		return reportName + " · no commits"
	}
	from := w.From.UTC().Format(historyDateLayout)
	to := w.To.UTC().Format(historyDateLayout)
	return reportName + " · last " + itoa(w.Commits) + " commits · " + from + " → " + to
}

// tabularBreakFor returns the 79- or 109-column break the existing renderers
// produce, honouring --no-hborder and --ci. Centralised here so every
// history renderer agrees with the language tables.
func tabularBreakFor(wide bool) string {
	if wide {
		return getTabularWideBreak()
	}
	return getTabularShortBreak()
}

// renderBar returns a unicode bar of the given cell width filled to ratio.
// Falls back to ASCII '#' when --ci is on or output is not a TTY (CSV-safe
// callers should not use this helper).
func renderBar(ratio float64, width int) string {
	if width <= 0 {
		return ""
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	if asciiOutput() {
		filled := int(ratio*float64(width) + 0.5)
		return strings.Repeat("#", filled) + strings.Repeat(" ", width-filled)
	}

	// 8 sub-cell levels using the Block Elements range U+2581..U+2588.
	const blocks = "▏▎▍▌▋▊▉█"
	total := ratio * float64(width)
	full := int(total)
	remainder := total - float64(full)
	var sb strings.Builder
	for i := 0; i < full; i++ {
		sb.WriteRune('█')
	}
	if full < width {
		idx := int(remainder * 8)
		if idx > 0 {
			runes := []rune(blocks)
			sb.WriteRune(runes[idx-1])
			full++
		}
	}
	for i := full; i < width; i++ {
		sb.WriteRune(' ')
	}
	return sb.String()
}

// renderSparkline downsamples series to the given cell width and renders it
// with U+2581..U+2587 spark characters (or ASCII when --ci / no TTY). The
// full block U+2588 is intentionally excluded so peak cells keep a 1-pixel
// gap at the top — without it, adjacent tall cells merge into a solid wall
// when trajectories rise monotonically.
// Used by plans 04–05; placed here so the helpers travel together.
func renderSparkline(series []float64, width int) string {
	if width <= 0 || len(series) == 0 {
		return ""
	}
	buckets := downsampleSeries(series, width)
	mx := 0.0
	for _, v := range buckets {
		if v > mx {
			mx = v
		}
	}
	if mx == 0 {
		if asciiOutput() {
			return strings.Repeat(".", width)
		}
		return strings.Repeat("▁", width)
	}

	if asciiOutput() {
		const ramp = " .:-=+*#%@"
		var sb strings.Builder
		for _, v := range buckets {
			idx := int(v / mx * float64(len(ramp)-1))
			sb.WriteByte(ramp[idx])
		}
		return sb.String()
	}

	const ticks = "▁▂▃▄▅▆▇"
	runes := []rune(ticks)
	var sb strings.Builder
	for _, v := range buckets {
		idx := int(v / mx * float64(len(runes)-1))
		sb.WriteRune(runes[idx])
	}
	return sb.String()
}

// downsampleSeries averages contiguous chunks of series into n buckets. If
// len(series) <= n it pads to n with the trailing value.
func downsampleSeries(series []float64, n int) []float64 {
	if n <= 0 {
		return nil
	}
	out := make([]float64, n)
	if len(series) == 0 {
		return out
	}
	if len(series) <= n {
		copy(out, series)
		for i := len(series); i < n; i++ {
			out[i] = series[len(series)-1]
		}
		return out
	}
	step := float64(len(series)) / float64(n)
	for i := 0; i < n; i++ {
		lo := int(float64(i) * step)
		hi := int(float64(i+1) * step)
		if hi > len(series) {
			hi = len(series)
		}
		if hi <= lo {
			hi = lo + 1
		}
		sum := 0.0
		for j := lo; j < hi; j++ {
			sum += series[j]
		}
		out[i] = sum / float64(hi-lo)
	}
	return out
}

// asciiOutput returns true when callers should avoid box/block glyphs
// (CI mode or non-TTY stdout).
func asciiOutput() bool {
	if Ci {
		return true
	}
	if FileOutput != "" {
		return true
	}
	return !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// itoa is a tiny strconv.Itoa wrapper kept local so this file doesn't pull
// strconv just for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// formatWindowComment builds the "# window: depth=… commits=… from=… to=…"
// comment line used by CSV outputs.
func formatWindowComment(w HistoryWindow) string {
	var depth string
	if w.Depth == 0 {
		depth = "all"
	} else {
		depth = itoa(w.Depth)
	}
	return "# window: depth=" + depth +
		" commits=" + itoa(w.Commits) +
		" from=" + formatWindowDate(w.From) +
		" to=" + formatWindowDate(w.To)
}

func formatWindowDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(historyDateLayout)
}
