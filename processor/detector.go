// SPDX-License-Identifier: MIT

package processor

import (
	"cmp"
	"errors"
	"slices"
	"strings"
)

var (
	errMissingShebang              = errors.New("missing shebang")
	errUnknownShebang              = errors.New("unknown shebang")
	errUnableToDetermineShebangCmd = errors.New("unable to determine shebang command")
)

// DetectLanguage detects a language based on the filename returns the language extension and error
func DetectLanguage(name string) ([]string, string) {
	extension := ""

	if len(AllowListExtensions) == 0 {
		// Check the full name for special languages such as xmake.lua, meson.build, ...
		lang, ok := FilenameToLanguage[strings.ToLower(name)]
		if ok {
			return []string{lang}, name
		}

		t := strings.Count(name, ".")
		// If there is no . in the filename or it starts with one then check if #!
		if t == 0 || (name[0] == '.' && t == 1) {
			printWarnF("possible #! file: %s", name)

			// No extension indicates possible #! so mark as such for processing
			return []string{SheBang}, name
		}
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
		extension = getExtension(extension)
		language = ExtensionToLanguage[extension]
	}

	return language, extension
}

// DetectSheBang given some content attempt to determine if it has a #! that maps to a known language and return the language
func DetectSheBang(content string) (string, error) {
	if !strings.HasPrefix(content, "#!") {
		return "", errMissingShebang
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
		if slices.Contains(v, cmd) {
			// detects both full path and env usage
			return k, nil
		}
	}

	return "", errUnknownShebang
}

func scanForSheBang(content []byte) (string, error) {
	state := 0
	lastSlash := 0

	candidate1 := ""
	candidate2 := ""

loop:
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

			// between last slash and here is the first candidate which is either env or Perl/PHP/Python etc..
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
			break loop
		}
	}

	switch {
	case candidate1 == "env":
		return candidate2, nil
	case candidate1 != "":
		return candidate1, nil
	}

	return "", errUnableToDetermineShebangCmd
}

type languageGuess struct {
	Name  string
	Count int
}

// DetermineLanguage given a filename, fallback language, possible languages and content make a guess to the type.
// If multiple possible it will guess based on keywords similar to how https://github.com/vmchale/polyglot does
func DetermineLanguage(filename string, fallbackLanguage string, possibleLanguages []string, content []byte) string {
	// If being called through an API it's possible nothing is set here and as
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
	if len(content) > 20_000 {
		toCheck = string(content)[:20_000]
	} else {
		toCheck = string(content)
	}

	primary := ""

	toSort := make([]languageGuess, 0, len(possibleLanguages))
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

		// if no features are found that means that this one is considered the primary
		// and as such the default fallback if we don't find a suitable number of matching
		// keywords
		// consider YAML files for example, where cloudformation files can also be YAML
		// YAML can have any form so it's not possible to say "this is a yaml file"
		// so we can only say "this is likely to be a cloudformation file", and as such
		// we need to handle a fallback case, which in this case is nothing
		if len(langFeatures.Keywords) == 0 {
			primary = lan
		}

		toSort = append(toSort, languageGuess{Name: lan, Count: count})
	}

	slices.SortFunc(toSort, func(a, b languageGuess) int {
		if order := cmp.Compare(b.Count, a.Count); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})

	// fmt.Println(toSort)
	// fmt.Println(possibleLanguages)
	// fmt.Println(primary, toSort[0].Name, toSort[0].Count)

	if primary != "" && len(toSort) != 0 {
		// OK at this point we have a primary, which means we want 3 or more matches to count as something else
		if toSort[0].Count < 3 {
			// we didn't find enough results, so lets return the primary in this case
			return primary
		}
	}

	printWarnF("guessing language %s for file %s", toSort[0].Name, filename)
	printTraceF("nanoseconds to guess language: %s: %d", filename, makeTimestampNano()-startTime)

	if len(toSort) != 0 {
		return toSort[0].Name
	}

	return fallbackLanguage
}
