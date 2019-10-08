package processor

import (
	"errors"
	"fmt"
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

func scanForSheBang(content []byte) (string, error) {

	state := 0
	lastSlash := 0
	startPos := 0

	for i := 0; i < len(content); i++ {
		switch {
		case state == 0: // Start where we look for / before changing state
			if content[i] == '/' {
				lastSlash = i
				state = 1
			}
		case state == 1: // Keep looking for / till we hit whitespace or end
			if content[i] == '/' {
				lastSlash = i
			}
			if isWhitespace(content[i]) {
				state = 2
			}
		case state == 2:
			if !isWhitespace(content[i]) {
				startPos = i
			}
		}
	}

	fmt.Println(startPos, lastSlash)

	return string(content[lastSlash+1:]), nil
}
