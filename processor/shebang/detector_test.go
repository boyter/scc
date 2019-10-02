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