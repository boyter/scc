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