package processor

import (
	"errors"
	"strings"
)

// Given some content attempt to determine if it has a #! that maps to a known language and return the language
func DetectSheBang(content string) (string, error) {
	ProcessConstants()

	if !strings.HasPrefix(content, "#!") {
		return "", errors.New("Missing #!")
	}

	for k, v := range ShebangLookup {
		for _, x := range v {
			// detects both full path and env usage
			if strings.Contains(content, "/"+x) || strings.Contains(content, " "+x) {
				return k, nil
			}
		}
	}

	return "", errors.New("Unknown #!")
}
