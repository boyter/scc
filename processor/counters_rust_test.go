// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"
)

// withoutCharLiterals runs fn with the one deliberate disagreement of the Rust
// counter turned off, so everything else can be held to exact agreement with
// the generic loop.
func withoutCharLiterals(t *testing.T, fn func()) {
	t.Helper()

	previous := rustCharLiterals
	rustCharLiterals = false
	defer func() { rustCharLiterals = previous }()

	fn()
}

// The shapes a Rust file is written in, each of which the two counters have to
// read the same way. The character literal is the one thing they must not, and
// it has its own test below.
func TestRustCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	withoutCharLiterals(t, func() {
		for _, test := range []struct {
			name    string
			content string
		}{
			{"empty", ""},
			{"one newline", "\n"},
			{"no trailing newline", "let x = 1;"},
			{"blank lines", "\n\n\nfn a() {}\n\n"},
			{"line comment", "// a comment\nfn a() {}\n"},
			{"doc comment", "/// a doc comment\nfn a() {}\n"},
			{"inner doc comment", "//! a module doc\nfn a() {}\n"},
			{"block comment", "/*\n a comment\n*/\nfn a() {}\n"},
			{"block comment nests", "/* outer /* inner */ still comment */\nlet x = 1;\n"},
			{"block comment nests over lines", "/* a\n/* b\n*/ c\n*/\nlet x = 1;\n"},
			{"block comment three deep", "/* a /* b /* c */ d */ e */\nlet x = 1;\n"},
			{"close touching reopen", "/* one *//* two */\nlet x = 1;\n"},

			{"plain string", "let s = \"hello\";\nlet x = 1;\n"},
			{"string holding a comment", "let s = \"// not a comment\";\nlet x = 1;\n"},
			{"string holding a block opener", "let s = \"/* not a comment\";\nlet x = 1;\n"},
			{"escaped quote in a string", "let s = \"a \\\" b\";\n// a comment\n"},
			{"escaped backslash then quote", "let s = \"a \\\\\";\n// a comment\n"},
			{"unterminated string", "let s = \"never closed\nlet x = 1;\n"},

			{"raw string", "let s = r\"a b\";\nlet x = 1;\n"},
			{"raw string holding a quote", "let s = r#\"a \" b\"#;\nlet x = 1;\n"},
			{"raw string holding a comment", "let s = r\"// not a comment\";\nlet x = 1;\n"},
			{"raw string holding a block opener", "let s = r\"/* not a comment\";\nlet x = 1;\n"},
			{"raw string over lines", "let s = r\"a\nb\nc\";\nlet x = 1;\n"},
			{"raw string wrapping blank lines", "let s = r\"\n\n\n\";\nlet x = 1;\n"},
			{"raw string ignores a backslash", "let s = r\"a \\\";\nlet x = 1;\n"},
			{"raw string two hashes", "let s = r##\"a \"# b\"##;\nlet x = 1;\n"},
			{"raw string eight hashes", "let s = r########\"x\"########;\nlet y = 2;\n"},
			{"raw string nine hashes is not one", "let s = r#########\"x\"#########;\nlet y = 2;\n"},
			{"byte raw string", "let s = br\"a b\";\nlet x = 1;\n"},
			{"byte raw string with hashes", "let s = br##\"a \" b\"##;\nlet x = 1;\n"},
			{"raw string opening a line", "r\"a\";\nlet x = 1;\n"},
			{"an r that is not a raw string", "let r = 1;\nlet x = 1;\n"},
			{"for followed by a quote", "for\"a\";\nlet x = 1;\n"},
			{"raw marker at end of file", "let s = r\""},
			{"raw marker and hash at end of file", "let s = r#\""},
			{"bare r at end of file", "let s = r"},

			{"byte string", "let b = b\"bytes\";\nlet x = 1;\n"},
			{"byte char literal", "let c = b'x';\n// a comment\n"},
			{"byte char literal holding a quote", "let c = b'\"';\n// a comment\n"},
			{"byte char at end of file", "let c = b'"},

			{"lifetime", "fn f<'a>(x: &'a str) {}\nlet x = 1;\n"},
			{"lifetime static", "let s: &'static str = \"a\";\n// a comment\n"},
			{"two lifetimes", "struct S<'a, 'b> { a: &'a u8, b: &'b u8 }\nlet x = 1;\n"},
			{"lifetime bound", "fn f<'a: 'b, 'b>() {}\nlet x = 1;\n"},

			{"complexity words", "if a { for b in c { while d {} } }\nmatch e { _ => {} }\n"},
			{"loop", "loop {\n    break;\n}\nlet x = 1;\n"},
			{"loop with a space", "loop \n{\n}\nlet x = 1;\n"},
			{"else", "if a {} else {}\nif b {} else if c {}\n"},
			{"complexity inside a word", "let iffy = 1;\nlet forth = 2;\nlet looped = 3;\nlet matched = 4;\n"},
			{"complexity operators", "let b = a || c && d != e == f;\n"},
			{"complexity in a comment", "// if for while loop match && ||\nlet x = 1;\n"},
			{"complexity in a string", "let s = \"if for while loop match\";\n"},
			{"complexity at line start", "for a in b {}\nwhile c {}\n|| d;\n&& e;\n"},

			{"postfix question", "let x = f()?;\nlet y = g()?;\n"},
			{"postfix question sized", "fn f<T: ?Sized>() {}\nlet x = 1;\n"},
			{"postfix question sized with a space", "fn f<T: ? Sized>() {}\nlet x = 1;\n"},
			{"postfix question sizedness", "let x = a?Sizedness;\n"},
			{"question at end of file", "let x = f()?"},
			{"question opening the file", "?"},
			{"question after whitespace only", "   ?\nlet x = 1;\n"},

			{"crlf", "let a = 1;\r\n// a comment\r\nlet b = 2;\r\n"},
			{"tabs and spaces", "\t\t// indented comment\n\t\tlet x = 1;\n"},
			{"quote at end of file", "let s = \""},
			{"slash at end of file", "let x = 1; /"},
			{"star slash outside a comment", "let x = a */ b;\n"},
			{"nul byte", "let x = 1;\x00\nlet y = 2;\n"},

			// rust-analyzer's own lexer fixtures are full of these. A line
			// holding a single quote puts a newline where the character should
			// be and the next line's quote where the closer should be, and
			// reading that as a literal swallows the line ending.
			{"a lone quote on its own line", "'hello'\n''\n'\n'\n'spam'\n"},
			{"two lone quotes", "'\n'\n"},
			{"lone quote then a comment", "'\n// a comment\n"},
			{"quote newline quote", "'\n'\nlet x = 1;\n"},
		} {
			fast, generic := countBothWays(t, "Rust", []byte(test.content))
			compareCounts(t, "Rust", test.name, fast, generic)
		}
	})
}

// M15, and the whole point of the counter. A character literal holding a byte
// the generic loop reads as a token is the one place the two are meant to
// disagree, and a lifetime is the thing that must not be mistaken for one.
func TestRustCharLiteralAgainstLifetime(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
		// What the counter must say, and what the generic loop says, so the
		// divergence is pinned both ways rather than merely tolerated.
		code, comment               int64
		genericCode, genericComment int64
	}{
		{
			name:    "a literal holding a double quote",
			content: "let c = '\"';\n// one\n// two\n",
			code:    1, comment: 2,
			genericCode: 3, genericComment: 0,
		},
		{
			// LineJudge 4010 exactly: the suite wants 1 code and 2 comment, and
			// the generic loop answers 3 code and 0 comment. The fixture under
			// examples/linejudge carries a provenance header and so counts more
			// comment lines than the case proper; this is the bare case.
			name:    "4010, a literal holding a double quote",
			content: "let c = '\"';\n// a comment\n// another comment\n",
			code:    1, comment: 2,
			genericCode: 3, genericComment: 0,
		},
		{
			// LineJudge 4020 exactly: the suite wants 3 code and 3 comment,
			// against the generic loop's 6 code and 0 comment.
			name:    "4020, two literals holding escapes",
			content: "let a = '\\'';\nlet b = '\"';\nlet c = 1;\n// one\n// two\n// three\n",
			code:    3, comment: 3,
			genericCode: 6, genericComment: 0,
		},
		{
			name:    "a literal holding an escaped quote",
			content: "let c = '\\'';\nlet d = '\"';\n// one\n",
			code:    2, comment: 1,
			genericCode: 3, genericComment: 0,
		},
		{
			name:    "a literal holding a backslash",
			content: "let c = '\\\\';\nlet d = '\"';\n// one\n",
			code:    2, comment: 1,
			genericCode: 3, genericComment: 0,
		},
		{
			name:    "a lifetime is not a literal",
			content: "fn f<'a>(x: &'a str) {}\n// one\n",
			code:    1, comment: 1,
			genericCode: 1, genericComment: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fast, generic := countBothWays(t, "Rust", []byte(test.content))

			if fast.Code != test.code || fast.Comment != test.comment {
				t.Errorf("counter got code=%d comment=%d, want %d/%d",
					fast.Code, fast.Comment, test.code, test.comment)
			}
			if generic.Code != test.genericCode || generic.Comment != test.genericComment {
				t.Errorf("the generic loop got code=%d comment=%d, want %d/%d. "+
					"A divergence is pinned both ways, so the generic loop moving is as much a change as the counter moving",
					generic.Code, generic.Comment, test.genericCode, test.genericComment)
			}
		})
	}
}

// 4050 is the regression test for M15, and it is the reason the fix cannot live
// in languages.json. Adding a plain ' quote there would fix 4010 and 4020 and
// break this, because a lifetime would open a string that never closes.
//
// The counter has to agree with the generic loop here, not diverge from it:
// both count the same, and the fix must not change that.
func TestRustLifetimeRegression(t *testing.T) {
	ProcessConstants()

	for _, content := range []string{
		"let s = \"it's a string\";\nlet t: &'static str = s;\n// a comment\n",
		"struct S<'a> { x: &'a str }\n// a comment\n",
		"fn f<'a, 'b: 'a>(x: &'a str, y: &'b str) {}\n// a comment\n",
		"let v: Vec<&'_ str> = vec![];\n// a comment\n",
	} {
		fast, generic := countBothWays(t, "Rust", []byte(content))
		compareCounts(t, "Rust", "lifetime regression", fast, generic)

		if fast.Comment != 1 {
			t.Errorf("a lifetime swallowed the comment under it: content %q counted %d comment lines, want 1",
				content, fast.Comment)
		}
	}
}

// Every Rust file of a real tree read both ways, the counts having to be
// identical once the one deliberate difference is taken out. Point
// SCC_DIFF_RUST_CORPUS at a checkout of something large.
func TestRustCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpusToggle(t, "Rust", "SCC_DIFF_RUST_CORPUS", ".rs", &rustCharLiterals, "char literals")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestRustCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestRustCounterAgreesOnTheCorpus(t)
}

// The raw string delimiter is a run of hashes read out of the file, so it is
// attacker controlled and bounded only by maxRustRawHashes. These walk the
// shapes a malformed one comes in, which must produce no panic and no
// disagreement.
func TestRustRawStringBounds(t *testing.T) {
	ProcessConstants()

	previous := SpecialisedCounters
	t.Cleanup(func() { SpecialisedCounters = previous })

	hashes := strings.Repeat("#", maxRustRawHashes)
	long := strings.Repeat("#", maxRustRawHashes+4)

	for _, content := range []string{
		`r`,
		`r"`,
		`r#`,
		`r#"`,
		`r` + hashes,
		`r` + hashes + `"`,
		`r` + long + `"`,
		`r` + long + `"x"` + long,
		`"` + hashes,
		hashes + `"`,
		`#r"`,
		`rr"`,
		`brr"`,
		`br`,
		`br"`,
		`b"`,
		`b'`,
		`'`,
		`''`,
		`'''`,
		`'\`,
		`'\'`,
		`'\''`,
		`'\u`,
		`'\u{`,
		`'\u{1F600`,
		`'\u{1F600}'`,
		`'\x`,
		`'\x41'`,
		`'é'`,
		`'😀'`,
		"r\"\n\"",
		"'\n'",
		`let s = r` + long + `"unterminated`,
		"\x00r\"",
		"r\x00\"",
	} {
		fast, generic := countBothWays(t, "Rust", []byte(content))
		compareCounts(t, "Rust", "raw string bounds "+content, fast, generic)
	}
}

func benchmarkRustCorpus(b *testing.B, specialised bool) {
	b.Helper()
	benchmarkCorpus(b, "Rust", "SCC_DIFF_RUST_CORPUS", ".rs", specialised)
}

func BenchmarkCountStatsRustCorpusGeneric(b *testing.B) {
	benchmarkRustCorpus(b, false)
}

func BenchmarkCountStatsRustCorpusSpecialised(b *testing.B) {
	benchmarkRustCorpus(b, true)
}

// The same corpus with complexity turned off, which is the ceiling the counting
// loop could reach if complexity cost nothing at all.
func BenchmarkCountStatsRustCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkRustCorpus(b, true)
}
