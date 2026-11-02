// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The shapes an LLVM IR file is written in, each of which the two counters have
// to read the same way. The sixteen complexity checks are where the attention
// goes: six of the seven anchors carry more than one check, and every one of
// those pairs has to be shown counting once and only once, both where it stands
// alone and where the longer check holds the shorter one.
func TestLLVMIRCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "%1 = add i32 %0, 1"},
		{"blank lines", "\n\n\ndefine void @f() {\n\n"},

		// The semicolon is the whole of the comment syntax. There is no block
		// comment at all, so a slash star is ordinary code.
		{"line comment", "; ModuleID = 'a'\n%1 = add i32 %0, 1\n"},
		{"comment after code", "%1 = add i32 %0, 1 ; a comment\n"},
		{"comment on its own line", "  ; indented comment\n"},
		{"slash star opens nothing", "/* still code */\n%1 = add i32 %0, 1\n"},
		{"slash slash opens nothing", "// still code\n%1 = add i32 %0, 1\n"},
		{"semicolon at end of file", "%1 = add i32 %0, 1 ;"},
		{"comment holding every check", "; br switch and or xor invoke resume llvm.loop\n"},

		// One quote, escaping with a backslash the way the generic loop assumes
		// of every language that does not say otherwise.
		{"plain string", "@a = constant [2 x i8] c\"ab\"\n"},
		{"string holding a comment", "@a = constant [1 x i8] c\"; not a comment\"\n"},
		{"string holding a check", "@a = constant [1 x i8] c\"br switch and or\"\n"},
		{"escaped quote in a string", "@a = constant [1 x i8] c\"a \\\" b\"\n; a comment\n"},
		{"escaped backslash then quote", "@a = constant [1 x i8] c\"a \\\\\"\n; a comment\n"},
		{"unterminated string", "@a = c\"never closed\n%1 = add i32 %0, 1\n"},
		{"quote at end of file", "@a = c\""},
		{"escaped quote opens nothing", "%1 = \\\"\n%2 = add i32 %1, 1\n"},
		{"quoted identifier", "define void @\"a b\"() {\n  ret void\n}\n"},
		{"string opening a line", "\"a\"\n%1 = add i32 %0, 1\n"},

		// The r anchor: or, xor, catchret and cleanupret.
		{"or", "%1 = or i32 %0, 1\n"},
		{"xor", "%1 = xor i32 %0, 1\n"},
		{"or inside a word", "%color = add i32 %0, 1\n%for = add i32 %0, 1\n"},
		{"xor does not also count or", "%1 = xor i32 %0, 1\n"},
		{"or and xor together", "%1 = or i32 %0, 1\n%2 = xor i32 %1, 2\n"},
		{"catchret", "  catchret from %cp to label %next\n"},
		{"cleanupret", "  cleanupret from %cp unwind to caller\n"},
		{"ret is not a check", "  ret void\n"},
		{"catchret inside a word", "%xcatchret = add i32 %0, 1\n"},
		{"cleanupret inside a word", "%xcleanupret = add i32 %0, 1\n"},

		// The b anchor: br, callbr and indirectbr.
		{"br", "  br label %next\n"},
		{"conditional br", "  br i1 %c, label %a, label %b\n"},
		{"callbr", "  callbr void asm \"\", \"\"()\n          to label %a []\n"},
		{"indirectbr", "  indirectbr i8* %a, [label %b, label %c]\n"},
		{"callbr does not also count br", "  callbr void @f()\n"},
		{"indirectbr does not also count br", "  indirectbr i8* %a, []\n"},
		{"br inside a word", "%abr = add i32 %0, 1\n%brx = add i32 %0, 1\n"},
		{"br opening a line", "br label %next\n"},
		{"br at end of file", "  br "},

		// The h anchor: shl, lshr and ashr.
		{"shl", "%1 = shl i32 %0, 1\n"},
		{"lshr", "%1 = lshr i32 %0, 1\n"},
		{"ashr", "%1 = ashr i32 %0, 1\n"},
		{"all three shifts", "%1 = shl i32 %0, 1\n%2 = lshr i32 %1, 1\n%3 = ashr i32 %2, 1\n"},
		{"shl inside a word", "%xshl = add i32 %0, 1\n"},
		{"shr with no prefix", "%1 = shr i32 %0, 1\n"},

		// The w anchor: switch and catchswitch.
		{"switch", "  switch i32 %0, label %d [\n    i32 1, label %a\n  ]\n"},
		{"catchswitch", "  %cs = catchswitch within none [label %h] unwind to caller\n"},
		{"catchswitch does not also count switch", "  %cs = catchswitch within none []\n"},
		{"switch inside a word", "%aswitch = add i32 %0, 1\n"},
		{"switch with no space", "  switch(\n"},

		// The m anchor: llvm.loop and resume.
		{"llvm.loop", "  br label %a, !llvm.loop !0\n"},
		{"llvm.loop has no trailing space", "  br label %a, !llvm.loop!0\n"},
		{"llvm.loop inside a string opens no check", "!0 = !{!\"llvm.loop\"}\n"},
		{"resume", "  resume { i8*, i32 } %e\n"},
		{"llvm.loop and resume together", "  br label %a, !llvm.loop !0\n  resume { i8*, i32 } %e\n"},
		{"llvm inside a word", "%allvm.loop = add i32 %0, 1\n"},
		{"resume inside a word", "%xresume = add i32 %0, 1\n"},
		{"llvm.dbg is not llvm.loop", "  call void @llvm.dbg.value()\n"},

		// The k and a anchors, which carry one check each.
		{"invoke", "  invoke void @f() to label %a unwind label %b\n"},
		{"invoke inside a word", "%xinvoke = add i32 %0, 1\n"},
		{"and", "%1 = and i32 %0, 1\n"},
		{"and inside a word", "%band = add i32 %0, 1\n%andx = add i32 %0, 1\n"},
		{"and opening a line", "and i32 %0, 1\n"},
		{"and at end of file", "%1 = and "},

		{"every check on one line", "br switch and or xor shl lshr ashr invoke resume callbr indirectbr catchret cleanupret catchswitch llvm.loop\n"},
		{"a real function", "define i32 @f(i32 %a) {\nentry:\n  %0 = and i32 %a, 1\n  %1 = icmp eq i32 %0, 0\n  br i1 %1, label %t, label %e\nt:\n  ret i32 1\ne:\n  ret i32 0\n}\n"},

		{"crlf line endings", "define void @f() {\r\n; a comment\r\n  ret void\r\n}\r\n"},
		{"tabs and spaces", "\t\t; indented comment\n\t\t%1 = add i32 %0, 1\n"},
		{"null byte", "%1 = add\x00 i32\n"},
	} {
		fast, generic := countBothWays(t, "LLVM IR", []byte(test.content))
		compareCounts(t, "LLVM IR", test.name, fast, generic)
	}
}

// Every LLVM IR file of a real tree read both ways. Point
// SCC_DIFF_LLVMIR_CORPUS at a checkout of llvm-project, which is where the
// bytes are: 720MB of .ll across 47,714 files, 39% of everything scc reads on
// that repository.
func TestLLVMIRCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "LLVM IR", "SCC_DIFF_LLVMIR_CORPUS", ".ll")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files. For LLVM IR the two tables are eleven bytes
// against four, so this is the larger of the two differences of any counter.
func TestLLVMIRCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestLLVMIRCounterAgreesOnTheCorpus(t)
}

func BenchmarkCountStatsLLVMIRCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "LLVM IR", "SCC_DIFF_LLVMIR_CORPUS", ".ll", false)
}

func BenchmarkCountStatsLLVMIRCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "LLVM IR", "SCC_DIFF_LLVMIR_CORPUS", ".ll", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsLLVMIRCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "LLVM IR", "SCC_DIFF_LLVMIR_CORPUS", ".ll", true)
}
