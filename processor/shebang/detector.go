package shebang

import (
	"errors"
	"strings"
)

/*
java,#!/opt/java/jdk-11/bin/java --source 11
bash,/bin/bash
dart,/usr/bin/env dart
fish,/bin/fish
groovy,/usr/bin/groovy
korn,/bin/ksh
lua,/usr/bin/env lua
ruby,/usr/bin/ruby
scala,/usr/bin/env scala
sed,usr/bin/sed
shell,/bin/sh
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
