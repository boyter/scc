package processor

import (
	"bytes"
	"strings"
	"testing"
)

func TestTraceLevel(t *testing.T) {
	if levelTrace.String() != "TRACE" {
		t.Errorf("traceLevel format error: got %s, want TRACE", levelTrace.String())
	}
	if levelDebug.String() != "DEBUG" {
		t.Errorf("traceLevel format error: got %s, want DEBUG", levelDebug.String())
	}
	if levelWarn.String() != "WARN" {
		t.Errorf("traceLevel format error: got %s, want WARN", levelWarn.String())
	}
	if levelError.String() != "ERROR" {
		t.Errorf("traceLevel format error: got %s, want ERROR", levelError.String())
	}
	if traceLevel(0).String() != "" {
		t.Error("formated unknown traceLevel is not empty")
	}
}

func TestDoPrint(t *testing.T) {
	testCases := []struct {
		level          traceLevel
		template       string
		args           []any
		expectedPrefix string
		expectedSuffix string
	}{
		{
			level:          levelTrace,
			template:       "test: %d",
			args:           []any{1},
			expectedPrefix: levelTrace.String(),
			expectedSuffix: ": test: 1\n",
		},
		{
			level:          levelWarn,
			template:       "test: %s",
			args:           []any{"message"},
			expectedPrefix: levelWarn.String(),
			expectedSuffix: ": test: message\n",
		},
		{
			level:          levelDebug,
			template:       "test",
			args:           nil,
			expectedPrefix: levelDebug.String(),
			expectedSuffix: ": test\n",
		},
	}
	buff := &bytes.Buffer{}
	for _, tc := range testCases {
		doPrint(buff, tc.level, tc.template, tc.args)
		if !strings.HasPrefix(buff.String(), tc.expectedPrefix) {
			t.Errorf("doPrint got \"%s\", want prefix \"%s\"", buff.String(), tc.expectedPrefix)
		}
		if !strings.HasSuffix(buff.String(), tc.expectedSuffix) {
			t.Errorf("doPrint got \"%s\", want suffix \"%s\"", buff.String(), tc.expectedSuffix)
		}
		buff.Reset()
	}
}

func TestPrintTrace(t *testing.T) {
	Trace = true
	printTrace("Testing print trace")
	Trace = false
	printTrace("Testing print trace")
}

func TestPrintDebug(t *testing.T) {
	Debug = true
	printDebug("Testing print debug")
	Debug = false
	printDebug("Testing print debug")
}

func TestPrintWarn(t *testing.T) {
	Verbose = true
	printWarn("Testing print warn")
	Verbose = false
	printWarn("Testing print warn")
}

func TestPrintError(t *testing.T) {
	printError("Testing print error")
}

func TestPrintWarnF(t *testing.T) {
	printWarnF("Testing print error")
}

func TestPrintDebugF(t *testing.T) {
	printDebugF("Testing print error")
}

func TestPrintTraceF(t *testing.T) {
	printTraceF("Testing print error")
}

func TestGetFormattedTime(t *testing.T) {
	res := getFormattedTime()

	if !strings.Contains(res, "T") {
		t.Error("String does not contain expected T", res)
	}

	if !strings.Contains(res, "Z") {
		t.Error("String does not contain expected Z", res)
	}
}
