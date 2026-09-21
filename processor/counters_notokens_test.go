// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"
)

// The shapes a file with no tokens in it is written in, each of which the fast
// path and the generic loop have to read the same way. The oracle is the
// generic loop, as it is for the eighteen counters, and it is the reason the
// trailing nul case below is here at all: it was written the obvious way first,
// and cpython's csv fuzz corpus disagreed.
func TestNoTokenPathAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one line of text", "hello\n"},
		{"no trailing newline", "hello"},
		{"a blank line", "hello\n\nworld\n"},
		{"leading blank lines", "\n\n\nhello\n"},
		{"spaces only is blank", "hello\n   \nworld\n"},
		{"tabs only is blank", "hello\n\t\t\nworld\n"},
		{"spaces and tabs only is blank", "hello\n \t \t\nworld\n"},
		{"indented text is code", "   hello\n"},
		{"carriage returns", "hello\r\nworld\r\n"},
		{"a bare carriage return line", "hello\n\r\nworld\n"},
		{"only newlines", "\n\n\n"},
		{"one newline", "\n"},
		{"a single space", " "},
		{"trailing blank line", "hello\n\n"},

		// Things that are tokens in other languages and are nothing here.
		{"a slash pair opens no comment", "// not a comment\n"},
		{"a hash opens no comment", "# not a comment\n"},
		{"a block comment opener is text", "/* still text */\n"},
		{"a quote opens no string", "\"unclosed\nstill text\n"},
		{"an apostrophe opens no string", "don't\nstill text\n"},
		{"a backslash splices nothing", "a line \\\nanother line\n"},
		{"complexity words are text", "if for while switch else\n"},

		// The nul. A file ending in one is not binary, because the generic
		// loop's state functions run to i < endPoint and never look at the last
		// byte. cpython carries exactly this, Modules/_xxtestfuzz, and a fast
		// path that checks the whole file drops a file the generic loop counts.
		{"a nul on the last byte", "a,b,c\n\n\x00"},
		{"a nul before the last byte", "a,b,c\n\x00\n"},
		{"a nul mid line", "a,\x00,c\nmore\n"},
		{"a nul on the first byte", "\x00abc\n"},

		// Past the ten thousand byte window a nul stops mattering to both.
		{"a nul past the binary window", strings.Repeat("filler line\n", 1200) + "x\x00y\nz\n"},

		{"a long blank line", strings.Repeat(" ", 4096) + "\n"},
		{"a long line of text", strings.Repeat("x", 4096) + "\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, language := range []string{"Plain Text", "Markdown", "JSON", "CSV"} {
				fast, generic := countBothWays(t, language, []byte(test.content))
				compareCounts(t, language, test.name+" ("+language+")", fast, generic)
			}
		})
	}
}

// The path is reached on a language declaring nothing, so what wants asserting
// is that ProcessMask being zero means exactly that and nothing else. Checked
// against languages.json itself rather than against a number: this test first
// counted the languages and held them to 32, which broke the day Txtar was
// added, and a count that has to be edited whenever the database grows is a
// count that will be edited without being thought about.
func TestNoTokenPathCoversTheLanguagesThatDeclareNothing(t *testing.T) {
	ProcessConstants()

	declaring := 0
	for name, language := range languageDatabase {
		LoadLanguageFeature(name)
		feature, ok := LanguageFeatures[name]
		if !ok {
			continue
		}

		declaresNothing := len(language.LineComment) == 0 &&
			len(language.MultiLine) == 0 &&
			len(language.Quotes) == 0 &&
			len(language.ComplexityChecks) == 0 &&
			len(language.ComplexityChecksPostfix) == 0

		if declaresNothing {
			declaring++
		}

		if noTokensAtAll(feature) != declaresNothing {
			t.Errorf("%s: the no-token path takes it %t, languages.json declares nothing %t "+
				"(line comments %d, block comments %d, quotes %d, checks %d/%d)",
				name, noTokensAtAll(feature), declaresNothing,
				len(language.LineComment), len(language.MultiLine), len(language.Quotes),
				len(language.ComplexityChecks), len(language.ComplexityChecksPostfix))
		}
	}

	// A set that has gone empty would satisfy every check above, so hold it to
	// having found some. The exact number is deliberately not asserted.
	if declaring < 20 {
		t.Errorf("only %d languages declare no tokens, which is too few to be right", declaring)
	}

	// The ones this was written for, so emptying the set still fails here.
	for _, want := range []string{"Plain Text", "Markdown", "JSON", "CSV", "ReStructuredText"} {
		feature, ok := LanguageFeatures[want]
		if !ok || !noTokensAtAll(feature) {
			t.Errorf("%s should reach the no-token path", want)
		}
	}
}
