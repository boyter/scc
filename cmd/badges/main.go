package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := processUrlPath(r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("you be invalid"))
			return
		}

		category := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("category")))

		title := "Total lines"

		switch category {
		case "code":
			title = "Code lines"
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
		default:
			//
			title = "Total lines"
		}

		textLength := "250"
		s := formatCount(30000)

		if len(s) <= 3 {
			textLength = "200"
		}

		w.Header().Set("Content-Type", "image/svg+xml;charset=utf-8")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="100" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h69v20H0z"/><path fill="#4c1" d="M69 0h31v20H69z"/><path fill="url(#b)" d="M0 0h100v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="355" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="590">` + title + `</text><text x="355" y="140" transform="scale(.1)" textLength="590">` + title + `</text><text x="835" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text><text x="835" y="140" transform="scale(.1)" textLength="` + textLength + `">` + s + `</text></g> </svg>`))
	})

	http.ListenAndServe(":8080", nil).Error()
}

type location struct {
	Location string
	User     string
	Repo     string
}

func (l *location) String() string {
	//url.Parse("https://"+l.Location)
	return ""
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
		return location{}, errors.New("")
	}

	return location{
		Location: s[0],
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

func process(id int, s string) {
	fmt.Println("processing", s)

	// Clean target just to be sure
	cmdArgs := []string{
		"-rf",
		"/tmp/scc-tmp-path-" + strconv.Itoa(id),
	}

	cmd := exec.Command("rm", cmdArgs...)
	err := cmd.Run()

	if err != nil {
		fmt.Println("rm start", err.Error())
		return
	}

	// Run git clone against the target
	// 180 seconds seems enough as the kernel itself takes about 60 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "git", "clone", "--depth=1", s+".git", "/tmp/scc-tmp-path-"+strconv.Itoa(id))

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	resp, err := cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		fmt.Println("git clone timed out")
		return
	}

	if err != nil {
		fmt.Println("git clone non-zero exit code", string(resp))
		return
	}

	// Run scc against what we just cloned
	fileName := processPath(s)

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
		fmt.Println("scc", err.Error())
	}

	//err = uploadS3File("sloccloccode", fileName, "/tmp/"+fileName)
	//if err != nil {
	//	fmt.Println("s3 upload", err.Error())
	//}
	//fmt.Println("uploaded now cleaning up")

	// Cleanup
	cmdArgs = []string{
		"-rf",
		"/tmp/" + fileName,
	}

	cmd = exec.Command("rm", cmdArgs...)
	err = cmd.Run()

	if err != nil {
		fmt.Println("rm cleanup filename", err.Error())
		return
	}

	cmdArgs = []string{
		"-rf",
		"/tmp/scc-tmp-path-" + strconv.Itoa(id),
	}

	cmd = exec.Command("rm", cmdArgs...)
	err = cmd.Run()

	if err != nil {
		fmt.Println("rm cleanup", err.Error())
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
		log.Fatal(err)
	}

	processedString := reg.ReplaceAllString(s, "")

	return processedString
}
