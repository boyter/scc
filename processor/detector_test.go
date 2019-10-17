package processor

import "testing"

func TestDetectSheBangEmpty(t *testing.T) {
	ProcessConstants()

	x, y := DetectSheBang("")

	if x != "" || y == nil {
		t.Error("Expected no match got", x)
	}
}

func TestDetectSheBangPerl(t *testing.T) {
	ProcessConstants()

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
			t.Error("Expected Perl match got", x, "for", c)
		}
	}
}

func TestDetectSheBangPhp(t *testing.T) {
	ProcessConstants()

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
	ProcessConstants()

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
	ProcessConstants()

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
	ProcessConstants()

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
	ProcessConstants()

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
	ProcessConstants()

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
	ProcessConstants()

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
	ProcessConstants()

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
	ProcessConstants()

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

func TestDetectSheBangShell(t *testing.T) {
	ProcessConstants()

	cases := []string{
		"#!/usr/bin/env sh",
		"#!/bin/sh",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Shell" || y != nil {
			t.Error("Expected Shell match got", x)
		}
	}
}

func TestDetectSheBangRuby(t *testing.T) {
	ProcessConstants()

	cases := []string{
		"#!/usr/bin/env ruby",
		"#!/usr/bin/ruby",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Ruby" || y != nil {
			t.Error("Expected Ruby match got", x)
		}
	}
}

func TestDetectSheBangLua(t *testing.T) {
	ProcessConstants()

	cases := []string{
		"#!/usr/bin/env lua",
		"#!/usr/bin/lua",
	}

	for _, c := range cases {
		x, y := DetectSheBang(c)

		if x != "Lua" || y != nil {
			t.Error("Expected Lua match got", x)
		}
	}
}

func TestDetectSheBangMultiple(t *testing.T) {
	ProcessConstants()

	x, y := DetectSheBang(`#!/python/perl/ruby`)

	if x != "Ruby" || y != nil {
		t.Error("Expected Ruby match got", x)
	}
}

func TestDetectSheBangMultipleNewLine(t *testing.T) {
	ProcessConstants()

	x, y := DetectSheBang(`#!/python/perl/ruby
python perl fish`)

	if x != "Ruby" || y != nil {
		t.Error("Expected Ruby match got", x)
	}
}

func TestScanSheBang(t *testing.T) {
	cases := []string{
		"#!/usr/bin/perl",
		"#!  /usr/bin/perl",
		"#!/usr/bin/perl -w",
		"#!/usr/bin/env perl",
		"#!  /usr/bin/env   perl",
		"#!/usr/bin/env perl -w",
		"#!  /usr/bin/env   perl   -w",
		"#!/opt/local/bin/perl",
	}

	for _, c := range cases {
		r, _ := scanForSheBang([]byte(c))

		if r != "perl" {
			t.Errorf("Expected 'perl' got '%s' for %s", r, c)
		}
	}
}

// Randomly try things to see what happens
func TestScanSheBangFuzz(t *testing.T) {
	for i := 0; i < 1000; i++ {
		x, _ := scanForSheBang([]byte(randStringBytes(100)))

		if x == "NEVERHAPPEN" {
			t.Errorf("Errr wot?")
		}
	}
}

func TestCheckFullNameSheBang(t *testing.T) {
	ProcessConstants()

	r, n := checkFullName("name")

	if n != "name" {
		t.Error("Expected name to return")
	}

	if r[0] != "#!" {
		t.Error("Expected #! return")
	}
}

func TestCheckFullNameLicense(t *testing.T) {
	ProcessConstants()

	r, n := checkFullName("license")

	if n != "license" {
		t.Error("Expected name to return")
	}

	if r[0] != "License" {
		t.Error("Expected License return")
	}
}

func TestGuessLanguageCoq(t *testing.T) {
	res := DetermineLanguage("", "", []string{"Coq", "SystemVerilog"}, []byte(`Require Hypothesis Inductive`))

	if res != "Coq" {
		t.Error("Expected guessed language to have been Coq got", res)
	}
}

func TestGuessLanguageSystemVerilog(t *testing.T) {
	res := DetermineLanguage("", "", []string{"Coq", "SystemVerilog"}, []byte(`endmodule posedge edge always wire`))

	if res != "SystemVerilog" {
		t.Error("Expected guessed language to have been SystemVerilog got", res)
	}
}

func TestGuessLanguageLanguageSetNoPossible(t *testing.T) {
	res := DetermineLanguage("", "Java", []string{}, []byte(`endmodule posedge edge always wire`))

	if res != "Java" {
		t.Error("Expected guessed language to have been Java got", res)
	}
}

func TestGuessLanguageSingleLanguageSet(t *testing.T) {
	res := DetermineLanguage("", "Java", []string{"Rust"}, []byte(`endmodule posedge edge always wire`))

	if res != "Rust" {
		t.Error("Expected guessed language to have been Rust got", res)
	}
}

func TestGuessLanguageLanguageEmptyContent(t *testing.T) {
	res := DetermineLanguage("", "", []string{"Rust"}, []byte(``))

	if res != "Rust" {
		t.Error("Expected guessed language to have been Rust got", res)
	}
}

// Benchmarks below

func BenchmarkScanSheBangFuzz(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = scanForSheBang([]byte(randStringBytes(100)))
	}
}

func BenchmarkScanSheBangReal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = scanForSheBang([]byte("#!  /usr/bin/env   perl   -w"))
	}
}
