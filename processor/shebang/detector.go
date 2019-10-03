package shebang

import (
	"errors"
	"strings"
)

/*
scala,/usr/bin/env scala
swift,/usr/bin/env swift
tcl,/usr/bin/env tcl
zsh,/bin/zsh
*/

var sheBangMatches = map[string][]string{
	"Perl":       {"perl"},
	"PHP":        {"php"},
	"Python":     {"python"},
	"AWK":        {"awk", "gawk", "mawk"},
	"C Shell":    {"csh", "tcsh"},
	"D":          {"rdmd"},
	"Erlang":     {"escript"},
	"JavaScript": {"node"},
	"Lisp":       {"sbcl"},
	"Scheme":     {"racket"},
	"Fish":       {"fish"},
	"BASH":       {"bash"},
	"Shell":      {"sh"},
	"Ruby":       {"ruby"},
	"Lua":        {"lua"},
	"Korn Shell": {"ksh"},
	"sed":        {"sed"},
	"TCL":        {"tcl"},
	"ZSH":        {"zsh"},
}

// Given some content attempt to determine if it has a #! that maps to a known language and return the language
func DetectSheBang(content string) (string, error) {
	if !strings.HasPrefix(content, "#!") {
		return "", errors.New("Missing #!")
	}

	for k, v := range sheBangMatches {
		for _, x := range v {
			// detects both full path and env usage
			if strings.Contains(content, "/"+x) || strings.Contains(content, " "+x) {
				return k, nil
			}
		}
	}

	return "", errors.New("Unknown #!")
}
