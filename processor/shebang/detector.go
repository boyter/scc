package shebang

import (
	"errors"
	"strings"
)

/*
perl,#!/usr/bin/perl
perl,#!  /usr/bin/perl
perl,#!/usr/bin/perl -w
perl,#!/usr/bin/env perl
perl,#!  /usr/bin/env   perl
perl,#!/usr/bin/env perl -w
perl,#!  /usr/bin/env   perl   -w
perl,#!/opt/local/bin/perl
perl,#!/usr/bin/perl5

php,#!/usr/bin/php
php,#!/usr/bin/php5

python,#!/usr/bin/python
python,#!/usr/bin/python2
python,#!/usr/bin/python3

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

func DetectSheBang(content string) (string, error) {
	if !strings.HasPrefix(content, "#!") {
		return "", errors.New("Missing #!")
	}

	if strings.Contains(content, "/perl") || strings.Contains(content, " perl") {
		return "Perl", nil
	}

	if strings.Contains(content, "/php") || strings.Contains(content, " php") {
		return "PHP", nil
	}

	return "", errors.New("Unknown #!")
}
