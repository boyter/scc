package processor

import (
	"errors"
	"strings"
)

// Given some content attempt to determine if it has a #! that maps to a known language and return the language
func DetectSheBang(content string) (string, error) {
	if !strings.HasPrefix(content, "#!") {
		return "", errors.New("Missing #!")
	}

	index := strings.Index(content, "\n")

	if index != -1 {
		content = content[:index]
	}

	cln := strings.Replace(content, " ", "", -1)
	cln = strings.Replace(cln, "-w", "", -1)

	for k, v := range ShebangLookup {
		for _, x := range v {
			// detects both full path and env usage
			if strings.HasSuffix(cln, "/"+x) || strings.Contains(content, " "+x) {
				return k, nil
			}
		}
	}

	return "", errors.New("Unknown #!")
}
