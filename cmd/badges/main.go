package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/boyter/scc/v3/processor"
	"github.com/boyter/simplecache"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

var (
	uniqueCode = "unique_code"
	cache      = simplecache.New[[]processor.LanguageSummary](simplecache.Option{
		MaxItems: intPtr(1000),
		MaxAge:   timePtr(time.Hour * 72),
	})
	countingSemaphore = make(chan bool, 1)
	tmpDir            = os.TempDir()
	json              = jsoniter.ConfigCompatibleWithStandardLibrary
	locationLog       = []string{}
	locationTracker   = map[string]int{}
	locationLogMutex  = sync.Mutex{}
)

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Duration) *time.Duration {
	return &t
}

func main() {
	http.HandleFunc("/health-check/", func(w http.ResponseWriter, r *http.Request) {
		locationLogMutex.Lock()
		for k, v := range locationTracker {
			_, _ = fmt.Fprintf(w, "%s:%d\n", k, v)
		}
		locationLogMutex.Unlock()
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loc, err := processUrlPath(r.URL.Path)
		if err != nil {
			http.Redirect(w, r, "https://github.com/boyter/scc/?tab=readme-ov-file#badges-beta", http.StatusTemporaryRedirect)
			return
		}

		if filterBad(loc) {
			log.Error().Str(uniqueCode, "bfee4bd8").Str("loc", loc.String()).Msg("filter bad")
			return
		}

		appendLocationLog(loc.String())

		res, err := process(1, loc)
		if err != nil {
			log.Error().Str(uniqueCode, "03ec75c3").Err(err).Str("loc", loc.String()).Send()
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("something bad happened sorry"))
			return
		}

		category := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("category")))
		wage := tryParseInt(strings.TrimSpace(strings.ToLower(r.URL.Query().Get("avg-wage"))), 56286)
		title, value := calculate(category, wage, res)

		if r.URL.Query().Get("lower") != "" {
			title = strings.ToLower(title)
		}

		s := formatCount(float64(value))
		if category == "effort" {
			s = formatMonthsInt(value)
		}

		textLength := "250"
		if len(s) <= 3 {
			textLength = "200"
		}

		bs := parseBadgeSettings(r.URL.Query())

		log.Info().Str(uniqueCode, "42c5269c").Str("loc", loc.String()).Str("category", category).Send()
		w.Header().Set("Content-Type", "image/svg+xml;charset=utf-8")
		if category == "effort" {
			_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="240" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="` + bs.TopShadowAccentColor + `" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="240" height="20" rx="3" fill="#` + bs.FontColor + `"/></clipPath><g clip-path="url(#a)"><path fill="#` + bs.TitleBackgroundColor + `" d="M0 0h125v20H0z"/><path fill="#` + bs.BadgeBackgroundColor + `" d="M125 0h115v20H125z"/><path fill="url(#b)" d="M0 0h240v20H0z"/></g><g fill="#` + bs.FontColor + `" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="625" y="150" fill="#` + bs.TextShadowColor + `" fill-opacity=".3" transform="scale(.1)">` + title + `</text><text x="625" y="140" transform="scale(.1)">` + title + `</text><text x="1825" y="150" fill="#` + bs.TextShadowColor + `" fill-opacity=".3" transform="scale(.1)">` + s + `</text><text x="1825" y="140" transform="scale(.1)">` + s + `</text></g> </svg>`))
		} else {
			_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="` + bs.TopShadowAccentColor + `" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="100" height="20" rx="3" fill="#` + bs.FontColor + `"/></clipPath><g clip-path="url(#a)"><path fill="#` + bs.TitleBackgroundColor + `" d="M0 0h69v20H0z"/><path fill="#` + bs.BadgeBackgroundColor + `" d="M69 0h31v20H69z"/><path fill="url(#b)" d="M0 0h100v20H0z"/></g><g fill="#` + bs.FontColor + `" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="355" y="150" fill="#` + bs.TextShadowColor + `" fill-opacity=".3" transform="scale(.1)" textLength="590">` + title + `</text><text x="355" y="140" transform="scale(.1)" textLength="590">` + title + `</text><text x="835" y="150" fill="#` + bs.TextShadowColor + `" fill-opacity=".3" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text><text x="835" y="140" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text></g> </svg>`))
		}
	})

	addr := ":8080"
	log.Info().Str(uniqueCode, "1876ce1e").Str("addr", addr).Msg("serving")
	if err := http.ListenAndServe(addr, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().Str(uniqueCode, "c28556e8").Err(err).Send()
		os.Exit(1)
	}
}

func filterBad(loc location) bool {
	l := loc.String()
	bad := []string{"wp-content.com", "wp-admin.com", ".well-known", "wp-includes.com", ".php"}
	count := 0
	for _, b := range bad {
		if strings.Contains(l, b) {
			count++
		}
	}

	return count >= 2
}

func appendLocationLog(log string) {
	locationLogMutex.Lock()
	defer locationLogMutex.Unlock()

	if slices.Contains(locationLog, log) {
		return
	}
	locationLog = append(locationLog, log)
	locationTracker[log] = locationTracker[log] + 1

	if len(locationLog) > 100 {
		locationLog = locationLog[1:]
	}
}

func calculate(category string, wage int, res []processor.LanguageSummary) (string, int64) {
	title := ""
	var value int64

	switch category {
	case "code", "codes":
		title = "Code lines"
		for _, x := range res {
			value += x.Code
		}
	case "blank", "blanks":
		title = "Blank lines"
		for _, x := range res {
			value += x.Blank
		}
	case "comment", "comments":
		title = "Comments"
		for _, x := range res {
			value += x.Comment
		}
	case "cocomo":
		title = "COCOMO $"
		for _, x := range res {
			value += x.Code
		}

		value = int64(estimateCost(value, wage))
	case "effort":
		title = "COCOMO Man Years"
		for _, x := range res {
			value += x.Code
		}

		value = int64(estimateScheduleMonths(value))
	case "line", "lines": // lines is the default
		fallthrough
	default:
		//
		title = "Total lines"
		for _, x := range res {
			value += x.Lines
		}
	}
	return title, value
}

type location struct {
	Provider string
	User     string
	Repo     string
}

func (l *location) String() string {
	loc := ".com/"
	ext := ".git"
	switch strings.ToLower(l.Provider) {
	case "bitbucket":
		loc = ".org/"
	case "git.sr.ht":
		loc = "/"
		ext = ""
	}

	parse, _ := url.Parse("https://" + l.Provider + loc + l.User + "/" + l.Repo + ext)
	return parse.String()
}

// processUrlPath takes in a string path and returns a struct
// that contains the location user and repo which is what most
// repositories need
// returns an error if we get anything other than 3 parts since thats
// the format we expect
func processUrlPath(path string) (location, error) {
	path = strings.ToLower(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	s := strings.Split(path, "/")
	if len(s) != 3 {
		return location{}, errors.New("invalid path part")
	}

	if s[0] == "sr.ht" {
		s[0] = "git.sr.ht"
	}

	return location{
		Provider: s[0],
		User:     s[1],
		Repo:     s[2],
	}, nil
}

type badgeSettings struct {
	FontColor            string
	TextShadowColor      string
	TopShadowAccentColor string
	TitleBackgroundColor string
	BadgeBackgroundColor string
}

// namedColors maps color names to their hex values (without #).
// Includes shields.io colors and common CSS color names for compatibility.
var namedColors = map[string]string{
	// shields.io specific colors
	"brightgreen": "44cc11",
	"green":       "97ca00",
	"yellowgreen": "a4a61d",
	"yellow":      "dfb317",
	"orange":      "fe7d37",
	"red":         "e05d44",
	"blue":        "007ec6",
	"lightgrey":   "9f9f9f",
	"lightgray":   "9f9f9f",
	"grey":        "555555",
	"gray":        "555555",
	"blueviolet":  "8a2be2",
	// shields.io semantic aliases
	"success":       "44cc11",
	"important":     "fe7d37",
	"critical":      "e05d44",
	"informational": "007ec6",
	"inactive":      "9f9f9f",
	// Common CSS color names
	"black":           "000000",
	"white":           "ffffff",
	"silver":          "c0c0c0",
	"maroon":          "800000",
	"purple":          "800080",
	"fuchsia":         "ff00ff",
	"lime":            "00ff00",
	"olive":           "808000",
	"navy":            "000080",
	"teal":            "008080",
	"aqua":            "00ffff",
	"cyan":            "00ffff",
	"magenta":         "ff00ff",
	"pink":            "ffc0cb",
	"coral":           "ff7f50",
	"salmon":          "fa8072",
	"gold":            "ffd700",
	"khaki":           "f0e68c",
	"violet":          "ee82ee",
	"indigo":          "4b0082",
	"crimson":         "dc143c",
	"turquoise":       "40e0d0",
	"tan":             "d2b48c",
	"brown":           "a52a2a",
	"chocolate":       "d2691e",
	"tomato":          "ff6347",
	"orchid":          "da70d6",
	"plum":            "dda0dd",
	"peru":            "cd853f",
	"sienna":          "a0522d",
	"beige":           "f5f5dc",
	"ivory":           "fffff0",
	"linen":           "faf0e6",
	"azure":           "f0ffff",
	"lavender":        "e6e6fa",
	"wheat":           "f5deb3",
	"snow":            "fffafa",
	"seashell":        "fff5ee",
	"honeydew":        "f0fff0",
	"mintcream":       "f5fffa",
	"aliceblue":       "f0f8ff",
	"ghostwhite":      "f8f8ff",
	"oldlace":         "fdf5e6",
	"papayawhip":      "ffefd5",
	"moccasin":        "ffe4b5",
	"bisque":          "ffe4c4",
	"mistyrose":       "ffe4e1",
	"lemonchiffon":    "fffacd",
	"cornsilk":        "fff8dc",
	"antiquewhite":    "faebd7",
	"floralwhite":     "fffaf0",
	"steelblue":       "4682b4",
	"royalblue":       "4169e1",
	"skyblue":         "87ceeb",
	"dodgerblue":      "1e90ff",
	"deepskyblue":     "00bfff",
	"cadetblue":       "5f9ea0",
	"cornflowerblue":  "6495ed",
	"mediumblue":      "0000cd",
	"darkblue":        "00008b",
	"midnightblue":    "191970",
	"slateblue":       "6a5acd",
	"darkslateblue":   "483d8b",
	"mediumslateblue": "7b68ee",
	"seagreen":        "2e8b57",
	"mediumseagreen":  "3cb371",
	"lightgreen":      "90ee90",
	"darkgreen":       "006400",
	"forestgreen":     "228b22",
	"limegreen":       "32cd32",
	"springgreen":     "00ff7f",
	"palegreen":       "98fb98",
	"darkseagreen":    "8fbc8f",
	"olivedrab":       "6b8e23",
	"darkolivegreen":  "556b2f",
	"darkred":         "8b0000",
	"firebrick":       "b22222",
	"indianred":       "cd5c5c",
	"lightsalmon":     "ffa07a",
	"darksalmon":      "e9967a",
	"lightcoral":      "f08080",
	"rosybrown":       "bc8f8f",
	"sandybrown":      "f4a460",
	"goldenrod":       "daa520",
	"darkgoldenrod":   "b8860b",
	"darkorange":      "ff8c00",
	"orangered":       "ff4500",
	"hotpink":         "ff69b4",
	"deeppink":        "ff1493",
	"palevioletred":   "db7093",
	"mediumvioletred": "c71585",
	"mediumpurple":    "9370db",
	"darkorchid":      "9932cc",
	"darkviolet":      "9400d3",
	"darkmagenta":     "8b008b",
	"slategray":       "708090",
	"slategrey":       "708090",
	"lightslategray":  "778899",
	"lightslategrey":  "778899",
	"darkslategray":   "2f4f4f",
	"darkslategrey":   "2f4f4f",
	"dimgray":         "696969",
	"dimgrey":         "696969",
	"darkgray":        "a9a9a9",
	"darkgrey":        "a9a9a9",
	"gainsboro":       "dcdcdc",
	"whitesmoke":      "f5f5f5",
}

// resolveColor converts a color input (name or hex) to a hex value without #.
// Returns the hex value if valid, or empty string if invalid.
func resolveColor(color string) string {
	color = strings.ToLower(color)

	// Check if it's a named color
	if hex, ok := namedColors[color]; ok {
		return hex
	}

	// Check if it's a valid hex color (3, 4, 6, or 8 digits)
	hexRegex := regexp.MustCompile(`^(?:(?:[\da-f]{3}){1,2}|(?:[\da-f]{4}){1,2})$`)
	if hexRegex.MatchString(color) {
		return color
	}

	return ""
}

// Parses badge settings from url query params
// if error, ignore and return default badge settings
func parseBadgeSettings(values url.Values) *badgeSettings {
	bs := badgeSettings{
		FontColor:            "fff",
		TextShadowColor:      "010101",
		TopShadowAccentColor: "bbb",
		TitleBackgroundColor: "555",
		BadgeBackgroundColor: "4c1",
	}

	fontColor := values.Get("font-color")
	textShadowColor := values.Get("font-shadow-color")
	topShadowAccentColor := values.Get("top-shadow-accent-color")
	titleBackgroundColor := values.Get("title-bg-color")
	badgeBackgroundColor := values.Get("badge-bg-color")

	// Resolve colors (supports both named colors and hex codes)
	if resolved := resolveColor(fontColor); resolved != "" {
		bs.FontColor = resolved
	}
	if resolved := resolveColor(textShadowColor); resolved != "" {
		bs.TextShadowColor = resolved
	}
	if resolved := resolveColor(topShadowAccentColor); resolved != "" {
		bs.TopShadowAccentColor = resolved
	}
	if resolved := resolveColor(titleBackgroundColor); resolved != "" {
		bs.TitleBackgroundColor = resolved
	}
	if resolved := resolveColor(badgeBackgroundColor); resolved != "" {
		bs.BadgeBackgroundColor = resolved
	}

	return &bs
}

// formatCount turns a float into a string usable for display
// to the user so, 2532 would be 2.5k and such up the various
// units
func formatCount(count float64) string {
	type r struct {
		val float64
		sym string
	}
	ranges := []r{
		{1e18, "E"},
		{1e15, "P"},
		{1e12, "T"},
		{1e9, "G"},
		{1e6, "M"},
		{1e3, "k"},
	}

	for _, v := range ranges {
		if count >= v.val {
			t := fmt.Sprintf("%.1f", math.Ceil(count/v.val*10)/10)

			if len(t) > 3 {
				t = t[:strings.Index(t, ".")]
			}

			return fmt.Sprintf("%v%v", t, v.sym)
		}
	}

	return fmt.Sprintf("%v", math.Round(count))
}

func process(id int, s location) ([]processor.LanguageSummary, error) {
	countingSemaphore <- true
	defer func() {
		<-countingSemaphore // remove one to free up concurrency
	}()

	val, ok := cache.Get(s.String())
	if ok {
		return val, nil
	}

	// Clean target just to be sure
	targetPath := filepath.Join(tmpDir, "scc-tmp-path-"+strconv.Itoa(id))
	if err := os.RemoveAll(targetPath); err != nil {
		return nil, err
	}

	// Run git clone against the target
	// 180 seconds seems enough as the kernel itself takes about 60 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", s.String(), targetPath)

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	_, err := cmd.Output()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	// Run scc against what we just cloned
	fileName := processPath(s.String())
	filePath := filepath.Join(tmpDir, fileName)

	if fileName == "" {
		return nil, errors.New("processPath returned empty")
	}

	cmdArgs := []string{
		"-f",
		"json",
		"-o",
		filePath,
		targetPath,
	}

	cmd = exec.Command("scc", cmdArgs...)
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var res []processor.LanguageSummary
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	_ = cache.Set(s.String(), res)

	// Cleanup
	if err := os.RemoveAll(filePath); err != nil {
		return nil, err
	}

	if err := os.RemoveAll(targetPath); err != nil {
		return nil, err
	}

	return res, nil
}

func processPath(s string) string {
	s = strings.ToLower(s)
	split := strings.Split(s, "/")

	if len(split) != 5 {
		return ""
	}

	sp := make([]string, 0, len(split))

	for _, s := range split {
		sp = append(sp, cleanString(s))
	}

	filename := strings.ReplaceAll(sp[2], ".com", "")
	filename = strings.ReplaceAll(filename, ".org", "")
	filename += "." + sp[3]
	filename += "." + strings.ReplaceAll(sp[4], ".git", "") + ".json"

	return filename
}

func cleanString(s string) string {
	reg, err := regexp.Compile("[^a-z0-9-._]+")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	processedString := reg.ReplaceAllString(s, "")

	return processedString
}

func estimateEffort(codeCount int64) float64 {
	return 3.2 * math.Pow(float64(codeCount)/1000, 1.05) * 1
}

func estimateCost(codeCount int64, averageWage int) float64 {
	return estimateEffort(codeCount) * (float64(averageWage) / 12) * 1.8
}

func estimateScheduleMonths(codeCount int64) float64 {
	return 2.5 * math.Pow(estimateEffort(codeCount), 0.38)
}

// FormatMonths converts a float64 representing months into a "X years Y months" string.
func formatMonths(totalMonths float64) string {
	years, fracMonths := math.Modf(totalMonths / 12)
	months := int(math.Round(fracMonths * 12))

	return fmt.Sprintf("%d years %d months", int(years), months)
}

// FormatMonthsInt converts an int64 representing months into a "X years Y months" string.
func formatMonthsInt(totalMonths int64) string {
	// Use integer division to get the number of years
	years := totalMonths / 12

	// Use the modulo operator to get the remaining months
	months := totalMonths % 12

	return fmt.Sprintf("%d years %d months", years, months)
}

func tryParseInt(s string, def int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}
