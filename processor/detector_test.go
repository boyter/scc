// SPDX-License-Identifier: MIT

package processor

import (
	"slices"
	"strings"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	ProcessConstants()
	AllowListExtensions = []string{"css"}
	_, ext := DetectLanguage("example.black.css")

	if ext != "css" {
		t.Error("Expected css got", ext)
	}
	AllowListExtensions = []string{}
}

func TestDetectLanguageMojoExtensions(t *testing.T) {
	ProcessConstants()

	tests := map[string]string{
		"example.mojo": "mojo",
		"example.🔥":    "🔥",
	}
	for filename, wantExtension := range tests {
		possible, extension := DetectLanguage(filename)
		if extension != wantExtension {
			t.Errorf("DetectLanguage(%q) extension = %q, want %q", filename, extension, wantExtension)
		}
		if !slices.Contains(possible, "Mojo") {
			t.Errorf("DetectLanguage(%q) languages = %v, want Mojo", filename, possible)
		}
	}
}

func TestDetectSheBangEmpty(t *testing.T) {
	ProcessConstants()

	x, y := DetectSheBang([]byte{})

	if x != "" || y == nil {
		t.Error("Expected no match got", x)
	}

	x, y = DetectSheBang(nil)

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

		if x != "Lisp" || y != nil {
			t.Error("Expected Lisp match got", x)
		}
	}
}

func TestDetectSheBangRacket(t *testing.T) {
	ProcessConstants()

	cases := []string{
		"#!/usr/bin/env racket",
		"#!/usr/bin/racket",
	}

	for _, c := range cases {
		x, y := DetectSheBang([]byte(c))

		if x != "Racket" || y != nil {
			t.Error("Expected Racket match got", x)
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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

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
		x, y := DetectSheBang([]byte(c))

		if x != "Lua" || y != nil {
			t.Error("Expected Lua match got", x)
		}
	}
}

func TestDetectSheBangMultiple(t *testing.T) {
	ProcessConstants()

	x, y := DetectSheBang([]byte(`#!/python/perl/ruby`))

	if x != "Ruby" || y != nil {
		t.Error("Expected Ruby match got", x)
	}
}

func TestDetectSheBangMultipleNewLine(t *testing.T) {
	ProcessConstants()

	data := `#!/python/perl/ruby
python perl fish`
	x, y := DetectSheBang([]byte(data))

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
	for range 1000 {
		x, _ := scanForSheBang([]byte(randStringBytes(100)))

		if x == "NEVERHAPPEN" {
			t.Errorf("Errr wot?")
		}
	}
}

func TestCheckFullNameSheBang(t *testing.T) {
	ProcessConstants()

	r, n := DetectLanguage("name")

	if n != "name" {
		t.Error("Expected name to return")
	}

	if r[0] != "#!" {
		t.Error("Expected #! return")
	}
}

func TestCheckFullNameLicense(t *testing.T) {
	ProcessConstants()

	r, n := DetectLanguage("license")

	if n != "license" {
		t.Error("Expected name to return")
	}

	if r[0] != "License" {
		t.Error("Expected License return")
	}
}

func TestCheckFullNameXMake(t *testing.T) {
	ProcessConstants()

	r, n := DetectLanguage("xmake.lua")

	if n != "xmake.lua" {
		t.Error("Expected xmake.lua to return")
	}

	if r[0] != "XMake" {
		t.Error("Expected XMake return")
	}

	// count xmake.lua as a lua file if AllowListExtensions was set
	AllowListExtensions = []string{"lua"}
	r, n = DetectLanguage("xmake.lua")

	if n != "lua" {
		t.Error("Expected lua to return")
	}

	if r[0] != "Lua" {
		t.Error("Expected Lua return")
	}
	AllowListExtensions = []string{}
}

func TestGuessLanguageCoq(t *testing.T) {
	ProcessConstants()

	res := DetermineLanguage("", "", []string{"Coq", "SystemVerilog"}, []byte(`Require Hypothesis Inductive`))

	if res != "Coq" {
		t.Error("Expected guessed language to have been Coq got", res)
	}
}

func TestGuessLanguageSystemVerilog(t *testing.T) {
	ProcessConstants()

	res := DetermineLanguage("", "", []string{"Coq", "SystemVerilog"}, []byte(`endmodule posedge edge always wire`))

	if res != "SystemVerilog" {
		t.Error("Expected guessed language to have been SystemVerilog got", res)
	}
}

func TestDetectLanguageIEC61131S7DCL(t *testing.T) {
	ProcessConstants()

	possible, ext := DetectLanguage("types.s7dcl")
	if ext != "s7dcl" {
		t.Error("Expected s7dcl got", ext)
	}
	found := slices.Contains(possible, "IEC61131-3")
	if !found {
		t.Error("Expected IEC61131-3 got", possible)
	}
}

func TestGuessLanguageIEC61131SCL(t *testing.T) {
	ProcessConstants()

	content := []byte(`FUNCTION_BLOCK "MotorControl"
VAR_INPUT
    Start : BOOL;
END_VAR
BEGIN
    IF Start THEN
        Speed := 100;
    END_IF;
END_FUNCTION_BLOCK`)

	res := DetermineLanguage("motor.scl", "", []string{"IEC61131-3", "Scallop"}, content)
	if res != "IEC61131-3" {
		t.Error("Expected guessed language to have been IEC61131-3 got", res)
	}
}

func TestGuessLanguageScallopSCL(t *testing.T) {
	ProcessConstants()

	content := []byte(`rel classes = {0, 1, 2}
rel count_enroll_cs_in_class(c, n) :-
  n = count(s: student(c, s), enroll(s, "CS") where c: classes(c))
query count_enroll_cs_in_class`)

	res := DetermineLanguage("scallop.scl", "", []string{"IEC61131-3", "Scallop"}, content)
	if res != "Scallop" {
		t.Error("Expected guessed language to have been Scallop got", res)
	}
}

// .h is shared between C / C++ / Objective-C. These exercise the regex
// heuristic disambiguation added for https://github.com/boyter/scc/issues/574

func TestDetectLanguageHeaderSharedExtension(t *testing.T) {
	ProcessConstants()

	possible, ext := DetectLanguage("foo.h")
	if ext != "h" {
		t.Error("Expected h got", ext)
	}
	for _, want := range []string{"C Header", "C++ Header", "Objective C"} {
		if !slices.Contains(possible, want) {
			t.Errorf("Expected %s among candidates got %v", want, possible)
		}
	}
}

func TestGuessLanguageHeaderCpp(t *testing.T) {
	ProcessConstants()

	content := []byte(`#ifndef EXAMPLE_CLASS
#define EXAMPLE_CLASS
class ExampleClass {
    public:
        ExampleClass();
    private:
        int MethodB();
};
#endif`)

	res := DetermineLanguage("example.h", "", []string{"C Header", "C++ Header", "Objective C"}, content)
	if res != "C++ Header" {
		t.Error("Expected guessed language to have been C++ Header got", res)
	}
}

func TestGuessLanguageHeaderObjectiveC(t *testing.T) {
	ProcessConstants()

	content := []byte(`#import <Foundation/Foundation.h>

@interface MyClass : NSObject
@property (nonatomic, strong) NSArray *items;
@end`)

	res := DetermineLanguage("myclass.h", "", []string{"C Header", "C++ Header", "Objective C"}, content)
	if res != "Objective C" {
		t.Error("Expected guessed language to have been Objective C got", res)
	}
}

func TestGuessLanguageHeaderPlainCFallback(t *testing.T) {
	ProcessConstants()

	content := []byte(`#ifndef FOO_H
#define FOO_H
int add(int a, int b);
void do_thing(void);
#endif`)

	res := DetermineLanguage("foo.h", "", []string{"C Header", "C++ Header", "Objective C"}, content)
	if res != "C Header" {
		t.Error("Expected guessed language to have been C Header got", res)
	}
}

// The result must not depend on the order candidates are supplied in, since
// ExtensionToLanguage is populated from a map with non-deterministic iteration.
func TestGuessLanguageHeaderDeterministic(t *testing.T) {
	ProcessConstants()

	content := []byte(`template <typename T>
class Foo {
    std::vector<T> items;
};`)

	orderings := [][]string{
		{"C Header", "C++ Header", "Objective C"},
		{"Objective C", "C++ Header", "C Header"},
		{"C++ Header", "Objective C", "C Header"},
	}
	for _, order := range orderings {
		res := DetermineLanguage("foo.h", "", order, content)
		if res != "C++ Header" {
			t.Errorf("Expected C++ Header for ordering %v got %s", order, res)
		}
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

func BenchmarkDetermineLanguage(b *testing.B) {
	ProcessConstants()

	coqContent := []byte("Require Hypothesis Inductive\n")
	systemVerilogContent := []byte("endmodule posedge edge always wire\n")
	largeCoqContent := []byte("Require Hypothesis Inductive\n" + strings.Repeat("x", 25_000))
	largeSystemVerilogContent := []byte("endmodule posedge edge always wire\n" + strings.Repeat("y", 25_000))
	possibleLanguages := []string{"Coq", "SystemVerilog"}

	benchmarks := []struct {
		name    string
		content []byte
	}{
		{name: "small_coq", content: coqContent},
		{name: "small_systemverilog", content: systemVerilogContent},
		{name: "large_coq_over_cutoff", content: largeCoqContent},
		{name: "large_systemverilog_over_cutoff", content: largeSystemVerilogContent},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(benchmark.content)))

			for i := 0; i < b.N; i++ {
				_ = DetermineLanguage("", "", possibleLanguages, benchmark.content)
			}
		})
	}
}
