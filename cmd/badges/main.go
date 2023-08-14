package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boyter/scc/v3/processor"
	"github.com/rs/zerolog/log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var uniqueCode = "unique_code"
var cache = NewSimpleCache(1000)
var countingSemaphore = make(chan bool, 1)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loc, err := processUrlPath(r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("you be invalid"))
			return
		}

		data, err := process(1, loc)
		if err != nil {
			log.Error().Str(uniqueCode, "03ec75c3").Err(err).Str("loc", loc.String()).Send()
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("something bad happened sorry"))
			return
		}

		var res []processor.LanguageSummary
		err = json.Unmarshal(data, &res)
		if err != nil {
			log.Error().Str(uniqueCode, "9192cad8").Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("something bad happened sorry"))
			return
		}

		category := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("category")))
		wage := tryParseInt(strings.TrimSpace(strings.ToLower(r.URL.Query().Get("avg-wage"))), 56286)
		title, value := calculate(category, wage, res)

		s := formatCount(float64(value))

		textLength := "250"
		if len(s) <= 3 {
			textLength = "200"
		}

		log.Info().Str(uniqueCode, "42c5269c").Str("loc", loc.String()).Str("category", category).Send()
		w.Header().Set("Content-Type", "image/svg+xml;charset=utf-8")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="100" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h69v20H0z"/><path fill="#4c1" d="M69 0h31v20H69z"/><path fill="url(#b)" d="M0 0h100v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="355" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="590">` + title + `</text><text x="355" y="140" transform="scale(.1)" textLength="590">` + title + `</text><text x="835" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text><text x="835" y="140" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text></g> </svg>`))
	})

	addr := ":8080"
	log.Info().Str(uniqueCode, "1876ce1e").Str("addr", addr).Msg("serving")
	http.ListenAndServe(addr, nil).Error()
}

func calculate(category string, wage int, res []processor.LanguageSummary) (string, int64) {
	title := ""
	var value int64

	switch category {
	case "codes":
		fallthrough
	case "code":
		title = "Code lines"
		for _, x := range res {
			value += x.Code
		}
	case "blank":
		fallthrough
	case "blanks":
		title = "Blank lines"
		for _, x := range res {
			value += x.Blank
		}
	case "comment":
		fallthrough
	case "comments":
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
	case "lines": // lines is the default
		fallthrough
	case "line": // lines is the default
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
	parse, _ := url.Parse("https://" + l.Provider + ".com/" + l.User + "/" + l.Repo + ".git")
	return parse.String()
}

// processUrlPath takes in a string path and returns a struct
// that contains the location user and repo which is what most
// repositories need
// returns an error if we get anything other than 3 parts since thats
// the format we expect
func processUrlPath(path string) (location, error) {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	s := strings.Split(path, "/")
	if len(s) != 3 {
		return location{}, errors.New("invalid path part")
	}

	return location{
		Provider: s[0],
		User:     s[1],
		Repo:     s[2],
	}, nil
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

func process(id int, s location) ([]byte, error) {
	countingSemaphore <- true
	defer func() {
		<-countingSemaphore // remove one to free up concurrency
	}()

	val, ok := cache.Get(s.String())
	if ok {
		return val, nil
	}

	// Clean target just to be sure
	cmdArgs := []string{
		"-rf",
		"/tmp/scc-tmp-path-" + strconv.Itoa(id),
	}

	cmd := exec.Command("rm", cmdArgs...)
	err := cmd.Run()

	if err != nil {
		return nil, err
	}

	// Run git clone against the target
	// 180 seconds seems enough as the kernel itself takes about 60 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "git", "clone", "--depth=1", s.String(), "/tmp/scc-tmp-path-"+strconv.Itoa(id))

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	_, err = cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	// Run scc against what we just cloned
	fileName := processPath(s.String())

	if fileName == "" {
		return nil, errors.New("processPath returned empty")
	}

	cmdArgs = []string{
		"-f",
		"json",
		"-o",
		"/tmp/" + fileName,
		"/tmp/scc-tmp-path-" + strconv.Itoa(id),
	}

	cmd = exec.Command("scc", cmdArgs...)
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile("/tmp/" + fileName)
	if err != nil {
		return nil, err
	}
	cache.Add(s.String(), file)

	// Cleanup
	cmdArgs = []string{
		"-rf",
		"/tmp/" + fileName,
	}

	cmd = exec.Command("rm", cmdArgs...)
	err = cmd.Run()

	if err != nil {
		return nil, err
	}

	cmdArgs = []string{
		"-rf",
		"/tmp/scc-tmp-path-" + strconv.Itoa(id),
	}

	cmd = exec.Command("rm", cmdArgs...)
	err = cmd.Run()

	if err != nil {
		return nil, err
	}

	return file, nil
}

func processPath(s string) string {
	s = strings.ToLower(s)
	split := strings.Split(s, "/")

	if len(split) != 5 {
		return ""
	}

	sp := []string{}

	for _, s := range split {
		sp = append(sp, cleanString(s))
	}

	filename := strings.Replace(sp[2], ".com", "", -1)
	filename = strings.Replace(filename, ".org", "", -1)
	filename += "." + sp[3]
	filename += "." + strings.Replace(sp[4], ".git", "", -1) + ".json"

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
