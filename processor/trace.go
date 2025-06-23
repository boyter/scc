package processor

import (
	"fmt"
	"io"
	"os"
	"time"
)

type traceLevel uint8

const (
	levelTrace traceLevel = iota + 1
	levelDebug
	levelWarn
	levelError
)

func (tl traceLevel) String() string {
	switch tl {
	case levelTrace:
		return "TRACE"
	case levelDebug:
		return "DEBUG"
	case levelWarn:
		return "WARN"
	case levelError:
		return "ERROR"
	default:
		return ""
	}
}

// Get the time as standard UTC/Zulu format
func getFormattedTime() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func prepareMsg(template string, args []any) string {
	if len(args) == 0 {
		return template
	}

	return fmt.Sprintf(template, args...)
}

func doPrint(dst io.Writer, level traceLevel, template string, args []any) {
	_, _ = fmt.Fprintf(dst, "%s %s: %s\n", level, getFormattedTime(), prepareMsg(template, args))
}

// Prints a message to stdout if flag to enable warning output is set
func printWarn(msg string) {
	if Verbose {
		doPrint(os.Stdout, levelWarn, msg, nil)
	}
}

// Prints a message to stdout if flag to enable warning output is set
func printWarnF(msg string, args ...any) {
	if Verbose {
		doPrint(os.Stdout, levelWarn, msg, args)
	}
}

// Prints a message to stdout if flag to enable debug output is set
func printDebug(msg string) {
	if Debug {
		doPrint(os.Stdout, levelDebug, msg, nil)
	}
}

// Prints a message to stdout if flag to enable debug output is set
func printDebugF(msg string, args ...any) {
	if Debug {
		doPrint(os.Stdout, levelDebug, msg, args)
	}
}

// Prints a message to stdout if flag to enable trace output is set
func printTrace(msg string) {
	if Trace {
		doPrint(os.Stdout, levelTrace, msg, nil)
	}
}

// Prints a message to stdout if flag to enable trace output is set
func printTraceF(msg string, args ...any) {
	if Trace {
		doPrint(os.Stdout, levelTrace, msg, args)
	}
}

// Used when explicitly for os.exit output when crashing out
func printError(msg string) {
	doPrint(os.Stderr, levelError, msg, nil)
}
