// SPDX-License-Identifier: MIT

package processor

import "testing"

// The hand written tables of counters_llvmir_test.go and counters_assembly_test.go
// compare the counter against the generic loop, which is the rule the eighteen
// are held to and is what catches a counter that has gone wrong. What it cannot
// catch is both loops being wrong together, and several of those cases carry a
// name that claims a number rather than an agreement: "catchswitch does not also
// count switch" reads as complexity 1, but it passes just as well if the pair of
// them answer 2.
//
// These pin the number. They are the claims where counting the token twice, or
// not at all, would be a silent wrong answer rather than a disagreement.
func TestCounterComplexityValues(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name     string
		language string
		content  string
		want     int64
	}{
		// A check that contains another check must count once, not twice.
		{"catchswitch is one check, not switch as well", "LLVM IR", "  %cs = catchswitch within none []\n", 1},
		{"callbr is one check, not br as well", "LLVM IR", "  callbr void @f()\n", 1},
		{"indirectbr is one check, not br as well", "LLVM IR", "  indirectbr i8* %a, []\n", 1},
		{"lshr is one check, not shr twice", "LLVM IR", "%1 = lshr i32 %0, 1\n", 1},
		{"xor does not also count or", "LLVM IR", "%1 = xor i32 %0, 1\n", 1},
		{"catchret is one check", "LLVM IR", "  catchret from %c to label %a\n", 1},
		{"cleanupret is one check", "LLVM IR", "  cleanupret from %c unwind to caller\n", 1},

		// A check spelled inside a longer word is not a check.
		{"and inside a word counts nothing", "LLVM IR", "%band = add i32 %0, 1\n%andx = add i32 %0, 1\n", 0},
		{"br inside a word counts nothing", "LLVM IR", "%abr = add i32 %0, 1\n%brx = add i32 %0, 1\n", 0},
		{"llvm.dbg is not llvm.loop", "LLVM IR", "  call void @llvm.dbg.value()\n", 0},
		{"llvm.loop inside a string counts nothing", "LLVM IR", "!0 = !{!\"llvm.loop\"}\n", 0},

		// llvm.loop carries no trailing space where br does.
		{"llvm.loop needs no trailing space", "LLVM IR", "  br label %a, !llvm.loop!0\n", 2},

		// Assembly spells switch, while and else with a trailing space only,
		// where C spells each of them twice. The bracket and brace forms are
		// therefore not checks here, and that is the whole reason the counter
		// takes spaceOpens rather than cOpens and braceOpens.
		{"while with a space is a check", "Assembly", "  while a\n", 1},
		{"while with a bracket is not", "Assembly", "  while(a)\n", 0},
		{"switch with a space is a check", "Assembly", "  switch a\n", 1},
		{"switch with a bracket is not", "Assembly", "  switch(a)\n", 0},
		{"else with a space is a check", "Assembly", "  else a\n", 1},
		{"else with a brace is not", "Assembly", "  else{a}\n", 0},
		{"if is a check either way", "Assembly", "  if a\n  if(a)\n", 2},
		{"for is a check either way", "Assembly", "  for a\n  for(a)\n", 2},
		{"a semicolon comment holds no check", "Assembly", "; if (a) while (b)\n", 0},
		{"a double slash is code, not a comment", "Assembly", "// if a\n", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, specialised := range []bool{false, true} {
				previous := SpecialisedCounters
				SpecialisedCounters = specialised
				job := FileJob{Language: test.language, Content: []byte(test.content), Bytes: int64(len(test.content))}
				CountStats(&job)
				SpecialisedCounters = previous

				loop := "generic loop"
				if specialised {
					loop = "counter"
				}
				if job.Complexity != test.want {
					t.Errorf("%s answered complexity %d, want %d", loop, job.Complexity, test.want)
				}
			}
		})
	}
}
