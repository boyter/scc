// SPDX-License-Identifier: MIT

package processor

import (
	"slices"
	"testing"
)

func TestIBMiLanguages(t *testing.T) {
	ProcessConstants()
	for filename, language := range map[string]string{
		"program.rpg": "RPG", "program.RPG": "RPG",
		"program.rpgle": "RPGLE", "program.RPGLE": "RPGLE",
		"program.sqlrpgle": "SQLRPGLE", "program.SQLRPGLE": "SQLRPGLE",
		"program.clp": "IBM i CL", "program.CLLE": "IBM i CL",
		"command.cmd": "IBM i CL", "command.CMD": "IBM i CL",
		"display.DSPF": "IBM i DDS", "physical.PF": "IBM i DDS", "logical.LF": "IBM i DDS",
		"printer.PRTF": "IBM i DDS", "menu.MNUDDS": "IBM i DDS", "command.MNUCMD": "IBM i DDS",
		"source.dds": "IBM i DDS",
	} {
		possible, _ := DetectLanguage(filename)
		if !slices.Contains(possible, language) {
			t.Errorf("DetectLanguage(%q) = %v, want %s", filename, possible, language)
		}
	}
}

func TestIBMiCommandDetection(t *testing.T) {
	ProcessConstants()
	for _, tc := range []struct{ content, language string }{
		{"/* IBM i command */\n CMD PROMPT('Example')\n PARM KWD(NAME) TYPE(*CHAR)\n", "IBM i CL"},
		{" cmd prompt('Example')\n", "IBM i CL"},
		{" CMD +\n PROMPT('Example')\n", "IBM i CL"},
		{"@echo off\nREM Windows command file\ncmd /c echo hello\n", "Batch"},
		{"echo CMD PROMPT(example)\n", "Batch"},
	} {
		possible, _ := DetectLanguage("example.cmd")
		if got := DetermineLanguage("example.cmd", "Batch", possible, []byte(tc.content)); got != tc.language {
			t.Errorf("DetermineLanguage(%q) = %s, want %s", tc.content, got, tc.language)
		}
	}
}

func TestIBMiLineCounts(t *testing.T) {
	tests := []struct {
		name, language, content string
		code, comment, blank    int64
	}{
		{"fixed RPG", "RPG", "00010 * comment\r\n00020C                   SETON                                        LR\r\n00030 \r\n", 1, 1, 1},
		{"short sequence lines", "RPG", "00010\n00020", 0, 0, 2},
		{"sequence tags", "RPGLE", "ju\nju     // comment\nB004 C                   EVAL      X = 1;", 1, 1, 1},
		{"compile-time data", "RPGLE", "     C                   SETON                                        LR\n**\n 003\n      * data, not a comment\n// also data\n'\n\n", 6, 0, 1},
		{"free compile-time data", "RPGLE", "**FREE\n// comment\n**CTDATA names\n// data\n**\n", 4, 1, 0},
		{"asterisk outside comment column", "RPG", "     C                   MULT      2\n     C                   EVAL      X = X * 2", 2, 0, 0},
		{"fixed RPGLE", "RPGLE", "     H DFTACTGRP(*NO)\n      * comment\n     C                   EVAL      X = X * 2\n", 2, 1, 0},
		{"mixed RPGLE", "RPGLE", "     H DFTACTGRP(*NO)\n      /free\n       // comment\n       x = x * 2; // trailing comment\n      /end-free\n      * fixed comment", 4, 2, 0},
		{"fully free", "RPGLE", "**FREE\n// comment\nx = x * 2;\nx = 'it''s // text';\n\n", 3, 1, 1},
		{"free column seven star", "RPGLE", "**free\n      *inlr = *on;\n", 2, 0, 0},
		{"BOM", "RPGLE", "\xef\xbb\xbf**FREE\n// comment", 1, 1, 0},
		{"embedded SQL", "SQLRPGLE", "**FREE\nexec sql\n  /* SQL comment\n     continued */\n  select 'it''s -- text' from sysibm.sysdummy1;\n// RPG comment\n-- SQL comment\n", 3, 4, 0},
		{"fixed SQL block", "SQLRPGLE", "       /* SQL comment\n      * still in block\n       continued */\n       exec sql select 1;\n", 1, 3, 0},
		{"CL strings and comments", "IBM i CL", "PGM\n/* comment\n   continued */\nSNDPGMMSG MSG('it''s /* text */')\n/* comment */ ENDPGM\n\n", 3, 2, 1},
		{"command source", "IBM i CL", "/* command\n   comment */\nCMD PROMPT('It''s /* text */')\nPARM KWD(NAME) TYPE(*CHAR)\n\n", 2, 2, 1},
		{"CL backslash does not escape", "IBM i CL", "SNDPGMMSG MSG('C:\\')\n/* comment */\n", 1, 1, 0},
		{"DDS", "IBM i DDS", "      * DDS comment\n     A          R RECORD\n     A            FIELD         10A\n\n", 2, 1, 1},
		{"DDS sequence numbers", "IBM i DDS", "00010A* comment\r\n00020A          R RECORD\r\n00030", 1, 1, 1},
		{"DDS comment markers in constants", "IBM i DDS", "     A                                  CONST('/* // *')\n     A                                  TEXT('It''s text')\n      * comment", 2, 1, 0},
		{"DDS short lines", "IBM i DDS", "\n     \n      *", 0, 1, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines, code, comment, blank := countOne(t, tc.language, tc.content)
			if code != tc.code || comment != tc.comment || blank != tc.blank || lines != code+comment+blank {
				t.Fatalf("got %d/%d/%d/%d, want %d/%d/%d/%d", lines, code, comment, blank, tc.code+tc.comment+tc.blank, tc.code, tc.comment, tc.blank)
			}
		})
	}
}
