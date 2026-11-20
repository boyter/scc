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
	line := strings.TrimSpace(strings.TrimPrefix(string(content), "#!"))
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", errUnableToDetermineShebangCmd
	}

	command := func(value string) string {
		if i := strings.LastIndexByte(value, '/'); i >= 0 {
			return value[i+1:]
		}
		return value
	}

	interpreter := command(fields[0])
	if interpreter != "env" {
		return interpreter, nil
	}

	for i := 1; i < len(fields); i++ {
		arg := fields[i]
		if arg == "--" || strings.HasPrefix(arg, "-") {
			// GNU env's -S/--split-string options may be written with the
			// command attached (-Spython3); the other options are flags.
			if strings.HasPrefix(arg, "-S") && len(arg) > 2 {
				return command(arg[2:]), nil
			}
			if strings.HasPrefix(arg, "--split-string=") {
				return command(strings.TrimPrefix(arg, "--split-string=")), nil
			}
			if arg == "-u" || arg == "-C" || arg == "--unset" || arg == "--chdir" || arg == "--argv0" {
				i++
			}
			continue
		}
		if strings.Contains(arg, "=") {
			// Environment assignments precede the command when using env.
			continue
		}
		return command(arg), nil
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
	var hitStack [planFoundStack]bool
	var hit []bool
	if len(plan.anchRe) <= planFoundStack {
		hit = hitStack[:len(plan.anchRe)]
	} else {
		hit = make([]bool, len(plan.anchRe))
	}
	plan.present(toCheck, found, hit)

	var best languageGuess

	for i, lan := range plan.langs {
		count := 0
		for _, h := range lan.heuristics {
			// An anchored heuristic was settled by the line walk, which tried
			// its pattern where the pattern could begin rather than searching
			// the file for it.
			if h.slot >= 0 {
				if hit[h.slot] {
					count++
				}
				continue
			}

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
