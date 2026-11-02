// SPDX-License-Identifier: MIT
package processor

import (
	"strings"
	"testing"
)

func TestIBMiComplexity(t *testing.T) {
	fixed := func(op string) string {
		return "     C" + strings.Repeat(" ", 19) + op + strings.Repeat(" ", max(0, 10-len(op))) + "X\n"
	}
	legacy := func(op string) string {
		return "     C" + strings.Repeat(" ", 21) + op + strings.Repeat(" ", max(0, 5-len(op))) + "X\n"
	}
	for _, tc := range []struct {
		name, language, content string
		want                    int64
	}{
		{"free decisions", "RPGLE", "**FREE\nif x; elseif y; endif;\nwhen x;\ndow x;\ndou x;\nfor i=1 to 10;\nendfor;\n", 6},
		{"case and extenders", "RPGLE", "**FREE\niF(e) x;\nDoW\tx;\n", 2},
		{"false positives", "RPGLE", "**FREE\n// IF WHEN FOR\nx='IF; WHEN';\nifname=1;\nendif;\nx = WHEN;\n**\nIF data\n", 0},
		{"fixed opcodes", "RPGLE", fixed("IFEQ") + fixed("DOWNE") + fixed("WHENLE") + fixed("CABGT") + fixed("CASLT") + fixed("IF(E)") + fixed("ENDIF") + fixed("DO"), 6},
		{"fixed operand", "RPGLE", "     C     IF            EVAL      X = WHEN\n     D IF              S             10A\n      * IF comment\n", 0},
		{"legacy opcode", "RPG", legacy("IFEQ") + legacy("DOUEQ") + legacy("ENDIF") + legacy("DO"), 2},
		{"legacy COMP", "RPG", legacy("COMP") + legacy("comp"), 2},
		{"RPGLE COMP", "RPGLE", fixed("COMP") + fixed("comp"), 2},
		{"SQLRPGLE COMP", "SQLRPGLE", fixed("COMP"), 1},
		{"COMP operand and comment", "RPGLE", "     C     COMP          EVAL      X = 1\n      * COMP comment\n", 0},
		{"free COMP identifier", "RPGLE", "**FREE\nCOMP = 1;\nx = 'COMP';\n", 0},
		{"mixed format", "RPGLE", fixed("IFEQ") + "      /free\n       dow x;\n       enddo;\n      /end-free\n" + fixed("ENDIF"), 2},
		{"CL decisions", "IBM i CL", "IF COND(&X) THEN(DO)\nWHEN COND(&X) THEN(DO)\nDOWHILE COND(&X)\nDOUNTIL COND(&X)\nDOFOR VAR(&I) FROM(1) TO(10)\nMONMSG MSGID(CPF0000)\nDO\nENDDO\nELSE\nENDSELECT\n", 6},
		{"CL labels and nested commands", "IBM i CL", "label: if COND(&X) THEN(IF COND(&Y))\n/* note */ dowhile COND(&X)\n", 3},
		{"CL parameters and comments", "IBM i CL", "/* IF MONMSG */\nCHGVAR VAR(&IF) VALUE('WHEN')\nCMD PROMPT('IF')\nPARM KWD(IF) TYPE(*CHAR)\nDO\nENDDO\n", 0},
		{"SQL and RPG", "SQLRPGLE", "**FREE\nexec sql select CASE WHEN X=1 THEN 1 END from T;\nif x;\n// IF\n", 1},
		{"multiline SQL", "SQLRPGLE", "**FREE\nexec sql select case\n when X=1 then 1\n end from T;\nif x;\n", 1},
		{"DDS", "IBM i DDS", "     A                                  TEXT('IF WHEN')\n", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ProcessConstants()
			job := &FileJob{Language: tc.language, Content: []byte(tc.content), Bytes: int64(len(tc.content)), TrackComplexityLines: true}
			CountStats(job)
			if job.Complexity != tc.want {
				t.Fatalf("got %d, want %d", job.Complexity, tc.want)
			}
			var total int64
			for _, v := range job.ComplexityLine {
				total += v
			}
			if total != job.Complexity {
				t.Fatalf("per-line count %d differs from total %d", total, job.Complexity)
			}
		})
	}
}

func TestFixedRPGComplexityDisabled(t *testing.T) {
	old := Complexity
	Complexity = true
	defer func() { Complexity = old; ProcessConstants() }()
	ProcessConstants()
	content := "     C" + strings.Repeat(" ", 19) + "IFEQ      X\n"
	job := &FileJob{Language: "RPGLE", Content: []byte(content), Bytes: int64(len(content))}
	CountStats(job)
	if job.Complexity != 0 {
		t.Fatalf("disabled complexity = %d", job.Complexity)
	}
}
