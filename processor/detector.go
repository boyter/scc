package processor

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Detects a language based on the filename returns the language extension and error
func DetectLanguage(name string) ([]string, string) {
	extension := ""

	t := strings.Count(name, ".")

	// If there is no . in the filename or it starts with one then check if #! or other
	if (t == 0 || (name[0] == '.' && t == 1)) && len(AllowListExtensions) == 0 {
		return checkFullName(name)
	}

	// Lookup in case the full name matches
	language, ok := ExtensionToLanguage[strings.ToLower(name)]

	// If no match check if we have a matching extension
	if !ok {
		extension = getExtension(name)
		language, ok = ExtensionToLanguage[extension]
	}

	// Convert from d.ts to ts and check that in case of multiple extensions
	if !ok {
		language, ok = ExtensionToLanguage[getExtension(extension)]
	}

	return language, extension
}

func checkFullName(name string) ([]string, string) {
	// Need to check if special type
	language, ok := FilenameToLanguage[strings.ToLower(name)]
	if ok {
		return []string{language}, name
	}

	if Verbose {
		printWarn(fmt.Sprintf("possible #! file: %s", name))
	}

	// No extension indicates possible #! so mark as such for processing
	return []string{SheBang}, name
}

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
		return "", err
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

	for i := range content {
		switch state {
		case 0: // Deals with whitespace after #! and before first /
			if content[i] == '/' {
				lastSlash = i
				state = 1
			}
		case 1: // Once we found the first / keep going till we hit whitespace
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
		case 2: // We have the first candidate, see if there is another
			// go till end of whitespace, mark that spot as new start
			if !isWhitespace(content[i]) {
				lastSlash = i
				state = 3
			}
		case 3:
			if i == len(content)-1 {
				candidate2 = string(content[lastSlash : i+1])
			}

			if isWhitespace(content[i]) {
				candidate2 = string(content[lastSlash:i])
				state = 4
			}
		case 4:
			break
		}
	}

	switch {
	case candidate1 == "env":
		return candidate2, nil
	case candidate1 != "":
		return candidate1, nil
	}

	return "", errors.New("Unable to determine #! command")
}

type languageGuess struct {
	Name  string
	Count int
}

// Given a filename, fallback language, possible languages and content make a guess to the type.
// If multiple possible it will guess based on keywords similar to how https://github.com/vmchale/polyglot does
func DetermineLanguage(filename string, fallbackLanguage string, possibleLanguages []string, content []byte) string {
	// If being called through an API its possible nothing is set here and as
	// such should just return as the Language value should have already been set
	if len(possibleLanguages) == 0 {
		return fallbackLanguage
	}

	// There should only be two possibilities now, either we have a single fallbackLanguage
	// in which case we set it and return
	// or we have multiple in which case we try to determine it heuristically
	if len(possibleLanguages) == 1 {
		return possibleLanguages[0]
	}

	startTime := makeTimestampNano()

	var toCheck string
	if len(content) > 20000 {
		toCheck = string(content)[:20000]
	} else {
		toCheck = string(content)
	}

	toSort := []languageGuess{}
	for _, lan := range possibleLanguages {
		LanguageFeaturesMutex.Lock()
		langFeatures := LanguageFeatures[lan]
		LanguageFeaturesMutex.Unlock()

		count := 0
		for _, key := range langFeatures.Keywords {
			if strings.Contains(toCheck, key) {
				count++
			}
		}

		toSort = append(toSort, languageGuess{Name: lan, Count: count})
	}

	sort.Slice(toSort, func(i, j int) bool {
		if toSort[i].Count == toSort[j].Count {
			return strings.Compare(toSort[i].Name, toSort[j].Name) < 0
		}

		return toSort[i].Count > toSort[j].Count
	})

	if Verbose {
		printWarn(fmt.Sprintf("guessing language %s for file %s", toSort[0].Name, filename))
	}

	if Trace {
		printTrace(fmt.Sprintf("nanoseconds to guess language: %s: %d", filename, makeTimestampNano()-startTime))
	}

	if len(toSort) != 0 {
		return toSort[0].Name
	}

	return fallbackLanguage
}
