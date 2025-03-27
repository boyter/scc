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

		textLength := "250"
		if len(s) <= 3 {
			textLength = "200"
		}

		bs := parseBadgeSettings(r.URL.Query())

		log.Info().Str(uniqueCode, "42c5269c").Str("loc", loc.String()).Str("category", category).Send()
		w.Header().Set("Content-Type", "image/svg+xml;charset=utf-8")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="` + bs.TopShadowAccentColor + `" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="100" height="20" rx="3" fill="#` + bs.FontColor + `"/></clipPath><g clip-path="url(#a)"><path fill="#` + bs.TitleBackgroundColor + `" d="M0 0h69v20H0z"/><path fill="#` + bs.BadgeBackgroundColor + `" d="M69 0h31v20H69z"/><path fill="url(#b)" d="M0 0h100v20H0z"/></g><g fill="#` + bs.FontColor + `" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="355" y="150" fill="#` + bs.TextShadowColor + `" fill-opacity=".3" transform="scale(.1)" textLength="590">` + title + `</text><text x="355" y="140" transform="scale(.1)" textLength="590">` + title + `</text><text x="835" y="150" fill="#` + bs.TextShadowColor + `" fill-opacity=".3" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text><text x="835" y="140" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text></g> </svg>`))
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

	fontColor := strings.ToLower(values.Get("font-color"))
	textShadowColor := strings.ToLower(values.Get("font-shadow-color"))
	topShadowAccentColor := strings.ToLower(values.Get("top-shadow-accent-color"))
	titleBackgroundColor := strings.ToLower(values.Get("title-bg-color"))
	badgeBackgroundColor := strings.ToLower(values.Get("badge-bg-color"))

	// Ensure valid colors
	r, err := regexp.Compile(`^(?:(?:[\da-f]{3}){1,2}|(?:[\da-f]{4}){1,2})$`)
	if err != nil {
		return &bs
	}
	if r.MatchString(fontColor) {
		bs.FontColor = fontColor
	}
	if r.MatchString(textShadowColor) {
		bs.TextShadowColor = textShadowColor
	}
	if r.MatchString(topShadowAccentColor) {
		bs.TopShadowAccentColor = topShadowAccentColor
	}
	if r.MatchString(titleBackgroundColor) {
		bs.TitleBackgroundColor = titleBackgroundColor
	}
	if r.MatchString(badgeBackgroundColor) {
		bs.BadgeBackgroundColor = badgeBackgroundColor
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

func tryParseInt(s string, def int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}
