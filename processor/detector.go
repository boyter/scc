// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
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
func DetectSheBang(content []byte) (string, error) {
	if !bytes.HasPrefix(content, []byte("#!")) {
		return "", errMissingShebang
	}

	content, _, _ = bytes.Cut(content, []byte{'\n'})

	cmd, err := scanForSheBang(content)
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

// planFoundStack is the size of the stack array used to hold which of a plan's
// literals a file contains. No language set comes close to it, and one that did
// would fall back to a heap slice rather than break.
const planFoundStack = 128

// better reports whether a guess beats the best so far: more matches wins, and
// the alphabetically first name breaks a tie so the choice is deterministic.
// Only the winner is ever read, so the candidates are compared as they are
// counted rather than collected into a slice and sorted.
func better(a, best languageGuess) bool {
	if a.Count != best.Count {
		return a.Count > best.Count
	}
	return a.Name < best.Name
}

func guessByHeuristics(filename string, plan *heuristicPlan, toCheck []byte) (string, bool) {
	if len(plan.langs) == 0 {
		return "", false
	}

	// One set of passes over the file answers every literal of every candidate,
	// rather than a pass per literal. See detector_plan.go.
	var stack [planFoundStack]bool
	var found []bool
	if plan.nlits <= planFoundStack {
		found = stack[:plan.nlits]
	} else {
		found = make([]bool, plan.nlits)
	}
	plan.present(toCheck, found)

	var best languageGuess

	for i, lan := range plan.langs {
		count := 0
		for _, h := range lan.heuristics {
			// Cheap necessary-literal pre-check so the expensive regex only runs
			// on the rare files that could actually match it.
			canMatch := len(h.ids) == 0
			for _, id := range h.ids {
				if found[id] {
					canMatch = true
					break
				}
			}
			if !canMatch {
				continue
			}
			if h.re.Match(toCheck) {
				count++
			}
		}

		guess := languageGuess{Name: lan.name, Count: count}
		if i == 0 || better(guess, best) {
			best = guess
		}
	}

	if best.Count > 0 {
		printWarnF("guessing language %s for file %s via heuristics", best.Name, filename)
		return best.Name, true
	}

	return "", false
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

	toCheck := content
	if len(content) > 20_000 {
		toCheck = content[:20_000]
	}

	// The candidate set settles which literals are worth looking for and which
	// languages the keyword count then has to weigh, and neither depends on the
	// file, so both are worked out once per set and held on the plan.
	plan := planFor(possibleLanguages)

	// First attempt regex heuristic disambiguation, used for shared extensions
	// such as .h between C / C++ / Objective-C.
	// Based on how linguist does it https://github.com/github-linguist/linguist/
	// which should be fine as its under MIT license
	if lang, ok := guessByHeuristics(filename, plan, toCheck); ok {
		printTraceF("nanoseconds to guess language: %s: %d", filename, makeTimestampNano()-startTime)
		return lang
	}

	// A language with heuristics of its own has had its say above, so what is
	// left is the ones settled by counting keywords. The primary among them --
	// the one with no keywords either -- is what a file falls back to when
	// nothing else is convincing: consider YAML files for example, where
	// cloudformation files can also be YAML. YAML can have any form so it's not
	// possible to say "this is a yaml file", only "this is likely to be a
	// cloudformation file", so there has to be a fallback.
	var best languageGuess
	for i, lan := range plan.fallbacks {
		count := 0
		for _, key := range lan.keywords {
			if bytes.Contains(toCheck, key) {
				count++
			}
		}

		guess := languageGuess{Name: lan.name, Count: count}
		if i == 0 || better(guess, best) {
			best = guess
		}
	}

	if len(plan.fallbacks) == 0 {
		return fallbackLanguage
	}

	if plan.primary != "" && best.Count < 3 {
		// OK at this point we have a primary, which means we want 3 or more
		// matches to count as something else, and we didn't find enough.
		return plan.primary
	}

	printWarnF("guessing language %s for file %s", best.Name, filename)
	printTraceF("nanoseconds to guess language: %s: %d", filename, makeTimestampNano()-startTime)

	return best.Name
}
