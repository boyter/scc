package main

import (
	"context"
	"errors"
	"fmt"
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

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loc, err := processUrlPath(r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("you be invalid"))
			return
		}

		process(1, loc)

		category := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("category")))

		title := ""

		switch category {
		case "codes":
			fallthrough
		case "code":
			title = "Code lines"
		case "blank":
			fallthrough
		case "blanks":
			title = "Blank lines"
		case "comment":
			fallthrough
		case "comments":
			title = "Comments"
		case "cocomo":
			title = "COCOMO $"
		case "lines": // lines is the default
			fallthrough
		case "line": // lines is the default
			fallthrough
		default:
			//
			title = "Total lines"
		}

		textLength := "250"
		s := formatCount(30000)

		if len(s) <= 3 {
			textLength = "200"
		}

		log.Info().Str(uniqueCode, "42c5269c").Str("loc", loc.String()).Str("category", category).Send()
		w.Header().Set("Content-Type", "image/svg+xml;charset=utf-8")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="100" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h69v20H0z"/><path fill="#4c1" d="M69 0h31v20H69z"/><path fill="url(#b)" d="M0 0h100v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="355" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="590">` + title + `</text><text x="355" y="140" transform="scale(.1)" textLength="590">` + title + `</text><text x="835" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text><text x="835" y="140" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text></g> </svg>`))
	})

	http.ListenAndServe(":8080", nil).Error()
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

func process(id int, s location) {
	_, ok := cache.Get(s.String())
	if ok {
		// TODO real thing here
		return
	}

	// Clean target just to be sure
	cmdArgs := []string{
		"-rf",
		"/tmp/scc-tmp-path-" + strconv.Itoa(id),
	}

	cmd := exec.Command("rm", cmdArgs...)
	err := cmd.Run()

	if err != nil {
		log.Error().Err(err).Str(uniqueCode, "41b5460d").Str("loc", s.String()).Send()
		return
	}

	// Run git clone against the target
	// 180 seconds seems enough as the kernel itself takes about 60 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "git", "clone", "--depth=1", s.String(), "/tmp/scc-tmp-path-"+strconv.Itoa(id))

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	resp, err := cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		log.Error().Str(uniqueCode, "8f8ccc64").Str("loc", s.String()).Msg("git clone timed out")
		return
	}

	if err != nil {
		log.Error().Err(err).Str(uniqueCode, "f28fb388").Str("loc", s.String()).Str("resp", string(resp)).Msg("git clone non-zero exit code")
		return
	}

	// Run scc against what we just cloned
	fileName := processPath(s.String())

	if fileName == "" {
		return
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
		log.Error().Err(err).Str(uniqueCode, "3a74fde3").Str("loc", s.String()).Str("resp", string(resp)).Send()
		return
	}

	file, err := os.ReadFile("/tmp/" + fileName)
	if err != nil {
		log.Error().Err(err).Str(uniqueCode, "b16570de").Str("loc", s.String()).Str("resp", string(resp)).Send()
		return
	}
	cache.Add(s.String(), file)
	//map[string]processor.LanguageSummary{}

	// Cleanup
	cmdArgs = []string{
		"-rf",
		"/tmp/" + fileName,
	}

	cmd = exec.Command("rm", cmdArgs...)
	err = cmd.Run()

	if err != nil {
		log.Error().Err(err).Str(uniqueCode, "ed40409c").Str("loc", s.String()).Str("resp", string(resp)).Send()
		return
	}

	cmdArgs = []string{
		"-rf",
		"/tmp/scc-tmp-path-" + strconv.Itoa(id),
	}

	cmd = exec.Command("rm", cmdArgs...)
	err = cmd.Run()

	if err != nil {
		log.Error().Err(err).Str(uniqueCode, "2bca46a1").Str("loc", s.String()).Str("resp", string(resp)).Send()
		return
	}
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
