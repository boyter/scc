package main

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//query := r.URL.Query().Get("q")
		//ext := r.URL.Query().Get("ext")
		fmt.Println(r.URL.Path)
		processPath(r.URL.Path)
		//pageSize := 20
		title := "Total lines"
		text_length := "250"
		s := formatCount(30000)

		w.Header().Set("Content-Type", "image/svg+xml;charset=utf-8")
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="100" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h69v20H0z"/><path fill="#4c1" d="M69 0h31v20H69z"/><path fill="url(#b)" d="M0 0h100v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="355" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="590">` + title + `</text><text x="355" y="140" transform="scale(.1)" textLength="590">` + title + `</text><text x="835" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="` + text_length + `">` + s + `</text><text x="835" y="140" transform="scale(.1)" textLength="` + text_length + `">` + s + `</text></g> </svg>`))
	})

	http.ListenAndServe(":8080", nil).Error()
}

type location struct {
	Location string
	User     string
	Repo     string
}

func processPath(path string) (location, error) {
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
