package shebang

import (
	"errors"
	"strings"
)

/*
awk,#!/usr/bin/awk
awk,#!/usr/bin/gawk
awk,#!/usr/bin/mawk

csh,#!/bin/csh
csh,#!/bin/tcsh

d,#!/usr/bin/env rdmd

erlang,#!/usr/bin/env escript
javascript,#!/usr/bin/env node
lisp,#!/usr/local/bin/sbcl
lisp,#!/usr/bin/env sbcl
scheme,#!/usr/bin/env racket

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

var SheBangMatches = map[string][]string{
	"Perl": {"perl"},
	"PHP": {"php"},
	"Python": {"python"},
}

func DetectSheBang(content string) (string, error) {
	if !strings.HasPrefix(content, "#!") {
		return "", errors.New("Missing #!")
	}

	for k, v := range SheBangMatches {
		for _, x := range v {
			if strings.Contains(content, "/" + x) || strings.Contains(content, " " + x) {
				return k, nil
			}
		}
	}

	return "", errors.New("Unknown #!")
}
