// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The shapes an Assembly file is written in, each of which the two counters
// have to read the same way. The double slash gets the attention: it opens a
// comment in every other language a counter is written for and opens nothing
// here, so a file full of // headers is a file full of code.
func TestAssemblyCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "  movq %rax, %rbx"},
		{"blank lines", "\n\n\n  ret\n\n"},

		// The semicolon, and the block comment Assembly shares with C.
		{"line comment", "; a comment\n  ret\n"},
		{"comment after code", "  ret ; a comment\n"},
		{"double slash is code", "// not a comment\n  ret\n"},
		{"double slash after code", "  ret // still code\n"},
		{"block comment over lines", "/*\n a comment\n*/\n  ret\n"},
		{"block comment closed then code", "/* a */ ret\n"},
		{"code then block comment", "  ret /* a\n b */\n  ret\n"},
		{"block comment does not nest", "/* outer /* inner */ still code */\n  ret\n"},
		{"close touching reopen", "/* one *//* two */\n  ret\n"},
		{"unterminated block comment", "/* never closed\n  ret\n"},
		{"slash at end of file", "  ret /"},
		{"star slash outside a comment", "  ret */ x\n"},
		{"semicolon inside a block comment", "/* ; still the comment */\n  ret\n"},

		// Two quotes, both escaping with a backslash.
		{"plain string", "  .ascii \"abc\"\n; a comment\n"},
		{"string holding a comment", "  .ascii \"; not a comment\"\n  ret\n"},
		{"string holding a block opener", "  .ascii \"/* not a comment\"\n  ret\n"},
		{"escaped quote in a string", "  .ascii \"a \\\" b\"\n; a comment\n"},
		{"escaped backslash then quote", "  .ascii \"a \\\\\"\n; a comment\n"},
		{"single quoted", "  .byte 'a'\n; a comment\n"},
		{"single quote holding a double", "  .byte '\"'\n; a comment\n"},
		{"lone single quote", "  .byte 'a\n  ret\n"},
		{"unterminated string", "  .ascii \"never closed\n  ret\n"},
		{"quote at end of file", "  .ascii \""},
		{"single quote at end of file", "  .byte '"},
		{"escaped quote opens nothing", "  x \\\"\n  ret\n"},
		{"apostrophe in a comment", "; it's a comment\n  ret\n"},

		// The eleven checks. switch, while and else are spelled with a space
		// and nothing else here, where C also writes them with a bracket or a
		// brace, so the bracket forms must not count.
		{"if with a space", "  if a\n"},
		{"if with a bracket", "  if(a)\n"},
		{"for with a space", "  for a\n"},
		{"for with a bracket", "  for(a)\n"},
		{"while with a space", "  while a\n"},
		{"while with a bracket is not a check", "  while(a)\n"},
		{"switch with a space", "  switch a\n"},
		{"switch with a bracket is not a check", "  switch(a)\n"},
		{"else with a space", "  else a\n"},
		{"else with a brace is not a check", "  else{a}\n"},
		{"operators", "  x = a || b && c != d == e\n"},
		{"complexity inside a word", "  retry\n  iffy\n  elsewhere\n  forward\n  meanwhile\n"},
		{"while does not also count else", "  while a\n"},
		{"switch does not also count while", "  switch a\n"},
		{"keyword opening a line", "for a\nwhile b\n|| c\n&& d\n"},
		{"complexity in a comment", "; if for while switch else || && != ==\n  ret\n"},
		{"complexity in a string", "  .ascii \"if for while switch else\"\n"},
		{"complexity in a block comment", "/* if for while switch else */\n  ret\n"},
		{"every check on one line", "if a for b while c switch d else e || f && g != h == i\n"},

		// Real-looking assembler, in the two dialects the corpus is full of.
		{"gas", "  .text\n  .globl main\nmain:\n  pushq %rbp\n  movq %rsp, %rbp\n  ret\n"},
		{"masm style", "_main PROC\n  mov eax, 1 ; return one\n  ret\n_main ENDP\n"},
		{"directive with a label", ".Lfunc_end0:\n  .size main, .Lfunc_end0-main\n"},

		{"crlf line endings", "  .text\r\n; a comment\r\n  ret\r\n"},
		{"tabs and spaces", "\t\t; indented comment\n\t\tret\n"},
		{"null byte", "  ret\x00 x\n"},
		{"backslash at end of line does not splice", "; a comment \\\n  ret\n"},
	} {
		fast, generic := countBothWays(t, "Assembly", []byte(test.content))
		compareCounts(t, "Assembly", test.name, fast, generic)
	}
}

// Every .s file of a real tree read both ways. Point SCC_DIFF_ASSEMBLY_CORPUS
// at a checkout of llvm-project, which holds 13,230 of them.
func TestAssemblyCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "Assembly", "SCC_DIFF_ASSEMBLY_CORPUS", ".s")
}

// The capital S is the preprocessed dialect, which is where the C macros and so
// the C shaped complexity checks actually appear. It is a different corpus and
// the extension sampled is case sensitive, so it wants a walk of its own.
func TestAssemblyCounterAgreesOnTheCapitalSCorpus(t *testing.T) {
	diffCorpus(t, "Assembly", "SCC_DIFF_ASSEMBLY_CORPUS", ".S")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestAssemblyCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestAssemblyCounterAgreesOnTheCorpus(t)
	TestAssemblyCounterAgreesOnTheCapitalSCorpus(t)
}

func BenchmarkCountStatsAssemblyCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "Assembly", "SCC_DIFF_ASSEMBLY_CORPUS", ".s", false)
}

func BenchmarkCountStatsAssemblyCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "Assembly", "SCC_DIFF_ASSEMBLY_CORPUS", ".s", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsAssemblyCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "Assembly", "SCC_DIFF_ASSEMBLY_CORPUS", ".s", true)
}
