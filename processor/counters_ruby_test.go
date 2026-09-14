// SPDX-License-Identifier: MIT

package processor

import (
	"strconv"
	"testing"
)

// The shapes a Ruby file is written in, each of which the two counters have to
// read the same way.
//
// The block comment gets most of the rows, since =begin and =end are the first
// multi-byte block comment delimiters of the sixteen and the only place Ruby
// asks for anything the earlier counters did not have. Several of these pin
// behaviour that is wrong about Ruby and right about the generic loop, which is
// the oracle: the delimiter needs no first column, and a heredoc is not a thing
// the loop knows about.
func TestRubyCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "x = 1"},
		{"blank lines", "\n\n\nx = 1\n\n"},
		{"line comment", "# a comment\nx = 1\n"},
		{"line comment after code", "x = 1 # trailing\ny = 2\n"},
		{"frozen string literal magic comment", "# frozen_string_literal: true\nx = 1\n"},

		// M11, the block comment.
		{"block comment at the first column", "=begin\ncomment\n=end\nx = 1\n"},
		{"block comment indented", "  =begin\n  comment\n  =end\nx = 1\n"},
		{"block comment after a code line", "x = 1\n=begin\ncomment\n=end\n"},
		{"block comment opened after code on the line", "x = 1 =begin\ncomment\n=end\ny = 2\n"},
		{"block comment closed mid line", "=begin\ncomment =end more\nx = 1\n"},
		{"closer with text run on", "=begin\ncomment\n=endx\nx = 1\n"},
		{"closer one byte short never closes", "=begin\ncomment\n=en\nx = 1\n"},
		{"opener with text run on still opens", "=beginx\ncomment\n=end\nx = 1\n"},
		{"opener one byte short opens nothing", "=begi\nx = 1\n"},
		{"closer with no opener", "=end\nx = 1\n"},
		{"empty block comment", "=begin\n=end\nx = 1\n"},
		{"block comment holding a hash", "=begin\n# not a line comment\n=end\nx = 1\n"},
		{"block comment holding a quote", "=begin\nit's fine\n=end\nx = 1\n"},
		{"block comment holding its own opener", "=begin\n=begin\n=end\nx = 1\n"},
		{"block comment wrapping blank lines", "=begin\n\n\n=end\nx = 1\n"},
		{"unterminated block comment", "=begin\nnever closed\n"},
		{"opener at end of file", "=begin"},
		{"opener and newline at end of file", "=begin\n"},
		{"equals begin inside a string", "s = \"=begin\"\nx = 1\n"},
		{"equals begin inside a line comment", "# =begin\nx = 1\n"},

		// The = collision. =begin, == and != all begin on an equals and the
		// counter stops on every one of them.
		{"assignment is not a check", "x = 1\ny = 2\n"},
		{"equality after an assignment", "x = 1\ny == 2\n"},
		{"double equals then begin opens a comment", "x ==begin\ncomment\n=end\ny = 1\n"},
		{"no space before begin opens a comment", "a=begin\ncomment\n=end\nb = 1\n"},
		{"bang equals then end is not a comment", "x !=end\ny = 1\n"},
		{"equality operators", "a != b\nc == d\ne || f\ng && h\n"},
		{"equals at end of file", "x ="},
		// The generic loop counts a check in the blank state and then steps the
		// cursor past it; the counter counts it from the code state one byte
		// later and steps over nothing. These are the runs where that could
		// double count or miss and does neither.
		{"equality opening a line", "== x\n"},
		{"triple equals opening a line", "=== x\n"},
		{"triple equals after code", "x === y\n"},
		{"quadruple equals", "==== x\n"},
		{"bang equals opening a line", "!= x\n"},
		{"or opening a line", "|| x\n"},
		{"and opening a line", "&& x\n"},
		{"run of pipes", "|||| x\n"},
		{"run of ampersands", "&&&& x\n"},
		{"two equalities on a line", "x == y == z\n"},
		{"equals begin truncated at end of file", "x =begi"},

		// Strings and quotes.
		{"escaped quote", "s = \"a \\\" b\"\n# a comment\n"},
		{"escaped backslash then quote", "s = \"a \\\\\"\n# a comment\n"},
		{"single quote holding a double", "s = 'a \" b'\nx = 1\n"},
		{"double quote holding a single", "s = \"it's\"\nx = 1\n"},
		{"interpolation", "s = \"a #{b} c\"\n# a comment\n"},
		{"hash inside a single quoted string", "s = 'a # b'\nx = 1\n"},
		{"percent w literal", "w = %w[one two three]\nx = 1\n"},
		{"unterminated string", "s = \"never closed\nx = 1\n"},
		{"quote at end of file", "s = \""},

		// The generic loop has no heredoc, so these are all code.
		{"heredoc", "x = <<~HEREDOC\n  text # not a comment\n  more\nHEREDOC\ny = 1\n"},
		{"heredoc holding a quote", "x = <<~SQL\n  it's here\nSQL\ny = 1\n"},
		{"heredoc holding a block opener", "x = <<~TEXT\n=begin\nTEXT\ny = 1\n"},

		// Complexity.
		{"complexity tokens", "if a\n  for b in c\n    while d\n    end\n  end\nelse \nend\n"},
		{"complexity inside a word", "retry = 1\niffy = 2\nelsewhere = 3\n"},
		{"complexity in a line comment", "# if for while && ||\nx = 1\n"},
		{"complexity in a block comment", "=begin\nif for while && ||\n=end\nx = 1\n"},
		{"complexity in a string", "s = \"if for while && ||\"\n"},
		{"switch with a bracket is not a check", "switch(y)\n"},
		{"else with a brace is not a check", "else{}\n"},
		{"begin spelled inside a word near else", "x = 1\n=belse \n"},

		{"crlf", "x = 1\r\n# a comment\r\ny = 2\r\n"},
		{"crlf block comment", "=begin\r\ncomment\r\n=end\r\nx = 1\r\n"},
		{"tabs and spaces", "\t\t# indented\n\t\tx = 1\n"},
		{"hash at end of file", "x = 1 #"},
	} {
		fast, generic := countBothWays(t, "Ruby", []byte(test.content))
		compareCounts(t, "Ruby", test.name, fast, generic)
	}
}

// The anchoring soundness argument of buildRubyStop, asserted rather than only
// written down.
//
// Ruby's complexity checks share =, e and i with its block comment delimiters,
// so a backwards read from an anchor could in principle walk into a delimiter
// and match something that is not there. The argument says it cannot, on the
// grounds that e and i are never anchors, that the interior of a delimiter is
// never scanned as code, and that neither delimiter ends in a byte any check
// carries behind its anchor.
//
// This walks every anchor byte placed immediately behind each delimiter, and
// the near misses the argument names by hand, and requires the generic loop to
// agree on all of them. A change that broke the argument would show up here
// rather than in a corpus nobody has.
func TestRubyAnchoringSurvivesTheDelimiterCollision(t *testing.T) {
	ProcessConstants()

	var cases []string
	for _, delimiter := range []string{"=begin", "=end"} {
		for _, anchor := range []string{"f", "w", "l", "=", "|", "&"} {
			cases = append(cases,
				delimiter+anchor+"\n",
				delimiter+anchor+" \n",
				"x "+delimiter+anchor+"\n",
			)
		}
	}

	// The shapes the argument calls out one at a time. =belse is the one that
	// gets furthest: its l really does have an e behind it, and only the word
	// boundary on the b in front stops the match.
	cases = append(cases,
		"=belse \n", "=belse x\n", "x =belse \n", "a=belse \n",
		"=bel\n", "=be lse \n", "=b else \n",
		"=endless\n", "=endl\n", "=ends \n",
		"=begine \n", "=beginl \n", "=beginf \n", "=beginw \n",
		"=end=\n", "=begin=\n", "=end= \n", "=begin= \n",
	)

	for _, content := range cases {
		fast, generic := countBothWays(t, "Ruby", []byte(content))
		compareCounts(t, "Ruby", strconv.Quote(content), fast, generic)
	}
}

// Every Ruby file of a real tree read both ways, the counts having to be
// identical. Point SCC_DIFF_RUBY_CORPUS at a checkout of something large.
func TestRubyCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "Ruby", "SCC_DIFF_RUBY_CORPUS", ".rb")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files. Ruby is the one language whose smaller table
// keeps an anchor byte, the = that opens =begin, so this is worth more here than
// elsewhere.
func TestRubyCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestRubyCounterAgreesOnTheCorpus(t)
}

// The block comment has to be found with complexity off as well, which is the
// one thing Ruby's smaller stop table has to keep that no other language's does.
func TestRubyBlockCommentSurvivesComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	content := []byte("=begin\ncomment\nmore\n=end\nx = 1\n")
	fast, generic := countBothWays(t, "Ruby", content)
	compareCounts(t, "Ruby", "block comment with complexity off", fast, generic)

	if fast.Comment != 4 {
		t.Errorf("counted %d comment lines with complexity off, want 4", fast.Comment)
	}
}

func BenchmarkCountStatsRubyCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "Ruby", "SCC_DIFF_RUBY_CORPUS", ".rb", false)
}

func BenchmarkCountStatsRubyCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "Ruby", "SCC_DIFF_RUBY_CORPUS", ".rb", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsRubyCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "Ruby", "SCC_DIFF_RUBY_CORPUS", ".rb", true)
}
