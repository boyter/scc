package shebang

import "testing"

func TestDetectSheBangEmpty(t *testing.T) {
	x, y := DetectSheBang("")

	if x != "" || y == nil {
		t.Error("Expected no match got", x)
	}
}

func TestDetectSheBangPerl(t *testing.T) {
	cases := []string{
		"#!/usr/bin/perl",
		"#!  /usr/bin/perl",
		"#!/usr/bin/perl -w",
		"#!/usr/bin/env perl",
		"#!  /usr/bin/env   perl",
		"#!/usr/bin/env perl -w",
		"#!  /usr/bin/env   perl   -w",
		"#!/opt/local/bin/perl",
		"#!/usr/bin/perl5",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Perl" || y != nil {
			t.Error("Expected Perl match got", x)
		}
	}
}

func TestDetectSheBangPhp(t *testing.T) {
	cases := []string{
		"#!/usr/bin/php5",
		"#!/usr/bin/php",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "PHP" || y != nil {
			t.Error("Expected PHP match got", x)
		}
	}
}

func TestDetectSheBangPython(t *testing.T) {
	cases := []string{
		"#!/usr/bin/python",
		"#!/usr/bin/python2",
		"#!/usr/bin/python3",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Python" || y != nil {
			t.Error("Expected Python match got", x)
		}
	}
}

func TestDetectSheBangAWK(t *testing.T) {
	cases := []string{
		"#!/usr/bin/awk",
		"#!/usr/bin/gawk",
		"#!/usr/bin/mawk",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "AWK" || y != nil {
			t.Error("Expected AWK match got", x)
		}
	}
}

func TestDetectSheBangCsh(t *testing.T) {
	cases := []string{
		"#!/bin/csh",
		"#!/bin/tcsh",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "C Shell" || y != nil {
			t.Error("Expected C Shell match got", x)
		}
	}
}

func TestDetectSheBangD(t *testing.T) {
	cases := []string{
		"#!/usr/bin/env rdmd",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "D" || y != nil {
			t.Error("Expected D match got", x)
		}
	}
}

func TestDetectSheBangNode(t *testing.T) {
	cases := []string{
		"#!/usr/bin/env node",
		"#!/usr/bin/node",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "JavaScript" || y != nil {
			t.Error("Expected JavaScript match got", x)
		}
	}
}

func TestDetectSheBangLisp(t *testing.T) {
	cases := []string{
		"#!/usr/bin/env sbcl",
		"#!/usr/bin/sbcl",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Lisp" || y != nil {
			t.Error("Expected Lisp match got", x)
		}
	}
}

func TestDetectSheBangScheme(t *testing.T) {
	cases := []string{
		"#!/usr/bin/env racket",
		"#!/usr/bin/racket",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Scheme" || y != nil {
			t.Error("Expected Scheme match got", x)
		}
	}
}

func TestDetectSheBangFish(t *testing.T) {
	cases := []string{
		"#!/usr/bin/env fish",
		"#!/usr/bin/fish",
		"#!/bin/fish",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Fish" || y != nil {
			t.Error("Expected Fish match got", x)
		}
	}
}
