// SPDX-License-Identifier: MIT

package processor

import "testing"

// The shapes a PHP file is written in, each of which the two counters have to
// read the same way. Both line comment tokens get their own rows, since PHP is
// the first language with two.
func TestPhpCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "$x = 1;"},
		{"open tag only", "<?php\n"},
		{"slash line comment", "// a comment\n$x = 1;\n"},
		{"hash line comment", "# a comment\n$x = 1;\n"},
		{"hash after code", "$x = 1; # trailing\n$y = 2;\n"},
		{"slash after code", "$x = 1; // trailing\n$y = 2;\n"},
		{"hash comment holding a slash comment", "# // not another\n$x = 1;\n"},
		{"slash comment holding a hash", "// # not another\n$x = 1;\n"},
		{"php 8 attribute reads as a comment", "#[Attribute]\nclass A {}\n"},
		{"hash inside a double quoted string", "$s = \"a # b\";\n$x = 1;\n"},
		{"hash inside a single quoted string", "$s = 'a # b';\n$x = 1;\n"},
		{"slashes inside a string", "$s = \"a // b\";\n$x = 1;\n"},
		{"block opener inside a string", "$s = \"a /* b\";\n$x = 1;\n"},
		{"block comment over lines", "/*\n a comment\n*/\n$x = 1;\n"},
		{"block comment does not nest", "/* outer /* inner */ still code */\n$x = 1;\n"},
		{"escaped quote", "$s = \"a \\\" b\";\n# a comment\n"},
		{"escaped backslash then quote", "$s = \"a \\\\\";\n# a comment\n"},
		{"single quote holding a double", "$s = 'a \" b';\n$x = 1;\n"},
		{"unterminated string", "$s = \"never closed\n$x = 1;\n"},
		{"unterminated block comment", "/* never closed\n$x = 1;\n"},
		{"complexity tokens", "if ($a) { for (;;) { while ($x) {} } }\nelse { switch ($y) {} }\n"},
		{"complexity inside a word", "$retry = 1;\n$iffy = 2;\n$elsewhere = 3;\n"},
		{"complexity operators", "$b = $a || $c && $d != $e == $f;\n"},
		{"complexity in a hash comment", "# if for while && ||\n$x = 1;\n"},
		{"complexity in a string", "$s = \"if for while && ||\";\n"},
		{"switch with a bracket is not a check", "switch($y) {}\n"},
		{"crlf", "$a = 1;\r\n# a comment\r\n$b = 2;\r\n"},
		{"tabs and spaces", "\t\t# indented\n\t\t$x = 1;\n"},
		{"quote at end of file", "$s = \""},
		{"hash at end of file", "$x = 1; #"},
		{"slash at end of file", "$x = 1; /"},
		{"html around the tags", "<html>\n<?php echo 1; ?>\n</html>\n"},
	} {
		fast, generic := countBothWays(t, "PHP", []byte(test.content))
		compareCounts(t, "PHP", test.name, fast, generic)
	}
}

// Every PHP file of a real tree read both ways, the counts having to be
// identical. Point SCC_DIFF_PHP_CORPUS at a checkout of something large.
func TestPhpCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "PHP", "SCC_DIFF_PHP_CORPUS", ".php")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestPhpCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestPhpCounterAgreesOnTheCorpus(t)
}

func BenchmarkCountStatsPhpCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "PHP", "SCC_DIFF_PHP_CORPUS", ".php", false)
}

func BenchmarkCountStatsPhpCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "PHP", "SCC_DIFF_PHP_CORPUS", ".php", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsPhpCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "PHP", "SCC_DIFF_PHP_CORPUS", ".php", true)
}
