// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"
)

// The shapes a Python file is written in, each of which the two loops have to
// read the same way. The docstring is the interesting one: the same three bytes
// are a comment when they open a line and a string when they follow code on it.
func TestPythonCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "x = 1"},
		{"blank lines", "\n\n\nx = 1\n\n"},
		{"hash comment", "# a comment\nx = 1\n"},
		{"hash inside a string", "x = \"# not a comment\"\ny = 2\n"},
		{"hash after code", "x = 1  # trailing\ny = 2\n"},

		// M7 and M10: the whole of Python's comment mechanism.
		{"docstring opening a line", "\"\"\"\ndoc\n\"\"\"\nx = 1\n"},
		{"docstring after code", "x = 1; \"\"\"\nnot doc\n\"\"\"\ny = 2\n"},
		{"docstring on one line", "\"\"\"doc\"\"\"\nx = 1\n"},
		{"docstring then code on the line", "\"\"\"doc\"\"\" ; x = 1\ny = 2\n"},
		{"docstring then a hash", "\"\"\"doc\"\"\" # c\nx = 1\n"},
		{"docstring then trailing spaces", "\"\"\"\ndoc\n\"\"\"   \nx = 1\n"},
		{"docstring wrapping blank lines", "\"\"\"\n\n\n\"\"\"\nx = 1\n"},
		{"string wrapping blank lines", "x = \"\"\"\n\n\n\"\"\"\ny = 1\n"},
		{"single quoted docstring", "'''\ndoc\n'''\nx = 1\n"},
		{"indented docstring", "def f():\n    \"\"\"\n    doc\n    \"\"\"\n    return 1\n"},
		{"empty docstring", "\"\"\"\"\"\"\nx = 1\n"},
		{"four quotes closing", "\"\"\"\ndoc\n\"\"\"\"\nx = 1\n"},
		{"five quotes closing", "\"\"\"\ndoc\n\"\"\"\"\"\nx = 1\n"},
		{"triple inside a docstring", "\"\"\"\na \"\"\" b \"\"\" c\n\"\"\"\nx = 1\n"},
		{"escaped closer in a docstring", "\"\"\"\ndoc\\\"\"\"\nstill\n\"\"\"\nx = 1\n"},
		{"escaped closer in a code triple", "x = \"\"\"\na\\\"\"\"\nb\n\"\"\"\ny = 1\n"},
		{"unterminated docstring", "\"\"\"\nnever closed\nx = 1\n"},
		{"docstring at end of file", "\"\"\""},
		{"docstring opener at end of file", "\"\"\"doc"},
		{"complexity inside a docstring", "\"\"\"\nif x:\nfor y:\n\"\"\"\nz = 1\n"},

		// M4 and M5: the prefixed forms.
		{"r docstring", "r\"\"\"\ndoc\n\"\"\"\nx = 1\n"},
		{"f docstring", "f\"\"\"\ndoc\n\"\"\"\nx = 1\n"},
		{"r single quoted docstring", "r'''\ndoc\n'''\nx = 1\n"},
		{"f single quoted docstring", "f'''\ndoc\n'''\nx = 1\n"},
		{"indented r docstring", "def f():\n    r\"\"\"\n    doc\n    \"\"\"\n    return 1\n"},
		{"r string", "x = r\"raw\\\"\n# c\n"},
		{"r string single quotes", "x = r'raw\\'\n# c\n"},
		{"f string is not a prefix form", "x = f\"hello {a}\"\n# c\n"},
		{"f string single quotes", "x = f'hello'\n# c\n"},
		{"b string is not a prefix form", "x = b\"bytes\"\n# c\n"},
		{"u string is not a prefix form", "x = u\"uni\"\n# c\n"},
		{"rb prefix", "x = rb\"a\"\n# c\n"},
		{"br prefix", "x = br\"a\"\n# c\n"},
		{"r inside an identifier", "foor\"x\"\n# c\n"},
		{"f inside an identifier", "deff\"\"\"\ndoc\n\"\"\"\n"},
		{"r docstring with six quotes", "r\"\"\"\"\"\"\nx = 1\n"},
		{"f docstring with six quotes", "f\"\"\"\"\"\"\nx = 1\n"},

		// Plain strings and escapes.
		{"escaped quote in a string", "x = \"a \\\" b\"\n# c\n"},
		{"escaped backslash then quote", "x = \"a \\\\\"\n# c\n"},
		{"quote behind a backslash", "x = \\\\\"a\"\n# c\n"},
		{"unterminated string", "x = \"never closed\ny = 1\n"},
		{"quote at end of file", "x = \""},
		{"lone quote", "\""},

		// Complexity, every check and every near miss.
		{"complexity words", "for x in y:\n    if a:\n        pass\n    elif b:\n        pass\n    else:\n        pass\n"},
		{"complexity while and with", "while x:\n    with open(f) as g:\n        pass\n"},
		{"complexity try family", "try:\n    pass\nexcept E:\n    pass\nfinally:\n    pass\n"},
		{"complexity match", "match x:\n    case 1:\n        pass\n"},
		{"complexity and or", "x = a and b or c\n"},
		{"complexity bracket forms", "for(x)\nif(a)\nelif(b)\nwhile(c)\nand(d)\nor(e)\nmatch(f)\n"},
		{"complexity with bracket", "with (a):\n"},
		{"complexity inside words", "format = 1\nelifer = 2\nwhiles = 3\nandy = 4\northo = 5\nmatcher = 6\ntrying = 7\nexcepting = 8\nfinallys = 9\n"},
		{"complexity at line start", "for x:\nwhile y:\nor z:\n"},
		{"complexity inside a string", "x = \"if for while and or\"\n"},
		{"else with colon and space", "else:\nelse x\n"},
		{"finally carries an l and an f", "finally:\n"},
		{"while carries an l and an h", "while x:\n"},

		// Whitespace, which Python has a great deal of.
		{"deep indentation", "                    x = 1\n"},
		{"tabs", "\t\t\tx = 1\n"},
		{"whitespace only line", "    \n\tx = 1\n"},
		{"crlf", "x = 1\r\n# c\r\n"},
		{"crlf docstring", "\"\"\"\r\ndoc\r\n\"\"\"\r\nx = 1\r\n"},
	} {
		fast, generic := countBothWays(t, "Python", []byte(test.content))
		compareCounts(t, "Python", test.name, fast, generic)
	}
}

func TestPythonCounterAgreesOnTheCorpus(t *testing.T) {
	ProcessConstants()
	diffCorpus(t, "Python", "SCC_DIFF_PYTHON_CORPUS", ".py")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestPythonCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestPythonCounterAgreesOnTheCorpus(t)
}

// --debug is the one surface the eligibility guard does not cover, because the
// generic loop's docstring state prints two lines deciding whether the line the
// closer sits on counts as comment or as code. The counter prints the same two,
// so turning the flag on does not quietly select a different loop. See spec 07
// 03-architecture §6, which recommends exactly this over widening the guard.
func TestPythonDebugOutputMatchesTheGenericLoop(t *testing.T) {
	ProcessConstants()

	previous := SpecialisedCounters
	t.Cleanup(func() { SpecialisedCounters = previous })

	Debug = true
	t.Cleanup(func() { Debug = false })

	for _, test := range []struct {
		name    string
		content string
	}{
		{"closer then newline", "\"\"\"\ndoc\n\"\"\"\nx = 1\n"},
		{"closer then code", "\"\"\"\ndoc\n\"\"\" x\ny = 1\n"},
		{"closer then a hash", "\"\"\"doc\"\"\" # c\nx = 1\n"},
		{"closer then spaces then code", "\"\"\"doc\"\"\"   y\nx = 1\n"},
		{"several docstrings", "\"\"\"a\"\"\"\n\"\"\"b\"\"\" c\n\"\"\"d\ne\"\"\"\n"},
		{"never closed", "\"\"\"\nnever\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			counter := capturingStdout(t, func() {
				job := FileJob{Language: "Python"}
				job.SetContent(test.content)
				SpecialisedCounters = true
				CountStats(&job)
			})

			generic := capturingStdout(t, func() {
				job := FileJob{Language: "Python"}
				job.SetContent(test.content)
				SpecialisedCounters = false
				CountStats(&job)
			})

			if counter != generic {
				t.Errorf("--debug output differs\n  counter: %q\n  generic: %q", counter, generic)
			}

			// Both lines begin the same way; only the one about a newline
			// mentions the docstring by name.
			if strings.Contains(test.name, "closer") && !strings.Contains(counter, "Found ") {
				t.Errorf("no docstring decision was logged at all: %q", counter)
			}
		})
	}
}

// A docstring that is never closed, and one whose closer is the last byte of the
// file, are the two shapes the bulk skip has to hand back correctly. Getting
// either wrong loses the trailing line, which is a wrong line count rather than
// a divergence.
func TestPythonDocStringRunsOut(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
		lines   int64
	}{
		{"never closed, no newline", "\"\"\"never", 1},
		{"never closed, one newline", "\"\"\"never\n", 1},
		{"never closed, many newlines", "\"\"\"a\nb\nc\nd\n", 4},
		{"closer is the last byte", "\"\"\"a\nb\"\"\"", 2},
		{"closer then newline at the end", "\"\"\"a\nb\"\"\"\n", 2},
		{"nothing but quotes", "\"\"\"\"\"\"\"\"\"", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			fast, generic := countBothWays(t, "Python", []byte(test.content))
			compareCounts(t, "Python", test.name, fast, generic)

			if fast.Lines != test.lines {
				t.Errorf("counted %d lines, want %d", fast.Lines, test.lines)
			}
		})
	}
}

func benchmarkPythonCorpus(b *testing.B, specialised bool) {
	b.Helper()
	benchmarkCorpus(b, "Python", "SCC_DIFF_PYTHON_CORPUS", ".py", specialised)
}

func BenchmarkCountStatsPythonCorpusGeneric(b *testing.B) {
	benchmarkPythonCorpus(b, false)
}

func BenchmarkCountStatsPythonCorpusSpecialised(b *testing.B) {
	benchmarkPythonCorpus(b, true)
}

// The ceiling the counting loop could reach if complexity cost nothing.
func BenchmarkCountStatsPythonCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkPythonCorpus(b, true)
}
