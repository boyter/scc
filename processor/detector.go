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

	cmd, err := scanForSheBang([]byte(content))

	if err != nil {
		return "", errors.New("Unable to determine #! command")
	}

	for k, v := range ShebangLookup {
		for _, x := range v {
			// detects both full path and env usage
			if x == cmd {
				return k, nil
			}
		}
	}

	return "", errors.New("Unknown #!")
}

func scanForSheBang(content []byte) (string, error) {
	state := 0
	lastSlash := 0

	candidate1 := ""
	candidate2 := ""

	for i := 0; i < len(content); i++ {
		switch {
		case state == 0: // Deals with whitespace after #! and before first /
			if content[i] == '/' {
				lastSlash = i
				state = 1
			}
		case state == 1: // Once we found the first / keep going till we hit whitespace
			if content[i] == '/' {
				lastSlash = i
			}

			// when at the end pull out the candidate
			if i == len(content)-1 {
				candidate1 = string(content[lastSlash+1 : i+1])
			}

			// between last slash and here is the first candidate which is either env or perl/php/python etc..
			if isWhitespace(content[i]) {
				// mark from lastSlash to here as first argument
				candidate1 = string(content[lastSlash+1 : i])
				state = 2
			}
		case state == 2: // We have the first candidate, see if there is another
			// go till end of whitespace, mark that spot as new start
			if !isWhitespace(content[i]) {
				lastSlash = i
				state = 3
			}
		case state == 3:
			if i == len(content)-1 {
				candidate2 = string(content[lastSlash : i+1])
			}

			if isWhitespace(content[i]) {
				candidate2 = string(content[lastSlash:i])
				state = 4
			}
		case state == 4:
			break
		}
	}

	switch {
	case candidate1 == "env":
		return candidate2, nil
	case candidate1 != "":
		return candidate1, nil
	}

	return "", errors.New("Count not find")
}
