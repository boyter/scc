// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

func countSwift(t *testing.T, content string) FileJob {
	t.Helper()
	ProcessConstants()
	fileJob := FileJob{
		Language: "Swift",
	}

	fileJob.SetContent(content)
	CountStats(&fileJob)

	return fileJob
}

func checkSwiftCounts(t *testing.T, fileJob FileJob, code, comment, blank int64) {
	t.Helper()
	if fileJob.Code != code {
		t.Errorf("Expected %d code got %d", code, fileJob.Code)
	}
	if fileJob.Comment != comment {
		t.Errorf("Expected %d comments got %d", comment, fileJob.Comment)
	}
	if fileJob.Blank != blank {
		t.Errorf("Expected %d blank got %d", blank, fileJob.Blank)
	}
}

// A raw string has no escape, so the backslash at the end of #"C:\"# does not
// carry the string on. Read as a plain string it ran on to the next quote in the
// file and turned the comments and blank lines in between into code.
func TestCountStatsSwiftRawString(t *testing.T) {
	fileJob := countSwift(t, `enum Paths {
    static let root = #"C:\"#

    // where the app is installed
    static let install = #"C:\Program Files\App"#

    /* used by the tests */
    static let fixtures = "Fixtures"
}`)

	if fileJob.Lines != 9 {
		t.Errorf("Expected 9 lines got %d", fileJob.Lines)
	}
	checkSwiftCounts(t, fileJob, 5, 2, 2)
}

// A raw string closes only on a quote followed by as many hashes as it opened
// with, so the quotes inside these do not end them.
func TestCountStatsSwiftRawStringHoldingQuotes(t *testing.T) {
	fileJob := countSwift(t, `let a = #"say "hi" "#
// a comment
let b = ##"a "# is not the end"##
/* another */
let c = 1`)

	checkSwiftCounts(t, fileJob, 3, 2, 0)
}

// Three quotes open a multi-line string, which was read as an empty string and
// a plain quote. The lone quote inside this one then opened a string that turned
// everything under it into code.
func TestCountStatsSwiftMultiLineString(t *testing.T) {
	fileJob := countSwift(t, `struct Laptop {
    let specs = """
        Display: 13.6" Liquid Retina
        // not a comment
        """

    // shown on the product page
    var summary: String { specs }

    /* one line per spec */
    var lines: [String] { specs.split(separator: "\n").map(String.init) }
}`)

	if fileJob.Lines != 12 {
		t.Errorf("Expected 12 lines got %d", fileJob.Lines)
	}
	checkSwiftCounts(t, fileJob, 8, 2, 2)
}

// A multi-line string keeps the backslash escape, so \""" does not close it.
func TestCountStatsSwiftMultiLineStringEscapedCloser(t *testing.T) {
	fileJob := countSwift(t, `let s = """
    a \""" in the middle
    // not a comment
    """
// a comment
let x = 1`)

	checkSwiftCounts(t, fileJob, 5, 1, 0)
}

// A multi-line raw string closes on """# and nothing shorter, so neither the
// lone quote nor the """ inside it ends it.
func TestCountStatsSwiftMultiLineRawString(t *testing.T) {
	fileJob := countSwift(t, `let source = #"""
    let s = "unterminated
    /* not a comment */
    """
    """#
// a comment

/* another */
print(source)`)

	checkSwiftCounts(t, fileJob, 6, 2, 1)
}

// #"""# is a raw string holding one quote, not the opener of a multi-line raw
// string, since its closer is on the same line.
func TestCountStatsSwiftRawStringOfOneQuote(t *testing.T) {
	fileJob := countSwift(t, `let quote = #"""#
// a comment

/* another */
let escaped = #"\""#
// a third`)

	checkSwiftCounts(t, fileJob, 2, 3, 1)
}

// A hash that is not in front of a quote opens nothing.
func TestCountStatsSwiftHashIsNotAQuote(t *testing.T) {
	fileJob := countSwift(t, `#if DEBUG
let action = #selector(tap)
#endif
// a comment
if #available(iOS 17, *) { }`)

	checkSwiftCounts(t, fileJob, 4, 1, 0)
}
