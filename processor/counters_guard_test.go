// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// The eligibility guard decides whether a counter may answer for a file at all,
// and until now no test asserted that any one of its seven conditions was
// honoured. A counter that quietly ran under --duplicates would corrupt the file
// hash and nothing would have caught it.
//
// Each condition below is turned on, the file is counted with the counters on
// and with them off, and the two have to agree on everything the flag produces.
// See spec 07 04-testing §4.

// guardSubject is a file with enough in it that a loop that skipped a state
// would show up: code, both comment forms, a string, complexity checks and
// blank lines.
const guardSubject = `class A {
    // a line comment
    /* a block
       comment over lines */

    void run() {
        if (a) { for (;;) {} }
        String s = "if for while";
        while (b) { try {} catch (E e) {} finally {} }
    }
}
`

// What each of the two runs wrote to stdout, which the trace case reads. Held
// here rather than returned so the shape of countWithGuard does not change for
// the six cases that do not want it.
var traceOffered, traceDeclined string

// countWithGuard counts the same content twice, once with the counters offered
// and once without, and hands back both jobs. Whatever the flag under test is,
// the guard has to send both runs down the generic loop and the two must match.
func countWithGuard(t *testing.T, prepare func(*FileJob)) (FileJob, FileJob) {
	t.Helper()

	previous := SpecialisedCounters
	t.Cleanup(func() { SpecialisedCounters = previous })

	offered := FileJob{Language: "Java", Filename: "A.java", Location: "A.java"}
	offered.SetContent(guardSubject)
	prepare(&offered)
	SpecialisedCounters = true
	traceOffered = capturingStdout(t, func() { CountStats(&offered) })

	declined := FileJob{Language: "Java", Filename: "A.java", Location: "A.java"}
	declined.SetContent(guardSubject)
	prepare(&declined)
	SpecialisedCounters = false
	traceDeclined = capturingStdout(t, func() { CountStats(&declined) })

	return offered, declined
}

// capturingStdout runs fn with os.Stdout pointed at a pipe and returns what it
// wrote. printTraceF writes to os.Stdout directly, so there is nowhere else to
// read the trace from.
func capturingStdout(t *testing.T, fn func()) string {
	t.Helper()

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not open a pipe: %v", err)
	}

	saved := os.Stdout
	os.Stdout = write

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, read)
		done <- buf.String()
	}()

	fn()

	os.Stdout = saved
	_ = write.Close()
	out := <-done
	_ = read.Close()

	return out
}

// countingCallback is the shape --history and blame ask for: a hook called once
// per line with what the line was counted as.
type countingCallback struct {
	lines []int64
}

func (c *countingCallback) ProcessLine(job *FileJob, line int64, lineType LineType) bool {
	c.lines = append(c.lines, int64(lineType))

	return true
}

func TestSpecialisedCounterGuardIsHonoured(t *testing.T) {
	ProcessConstants()

	// Every one of these is a thing the generic loop does inside its own state
	// functions that no counter replicates, so a counter must decline the file.
	for _, test := range []struct {
		name    string
		prepare func(*FileJob)
		restore func()
		// capture reads what the run wrote to stdout, which is where the trace
		// log goes and the only place a counter running under --trace would show.
		capture bool
		// extra checks the flag's own output, which is the part a plain line
		// count comparison would not notice.
		extra func(t *testing.T, offered, declined FileJob)
	}{
		{
			name:    "history",
			prepare: func(job *FileJob) { job.Callback = &countingCallback{} },
			extra: func(t *testing.T, offered, declined FileJob) {
				got := offered.Callback.(*countingCallback)
				want := declined.Callback.(*countingCallback)
				if len(got.lines) == 0 {
					t.Error("the per line callback was never called")
				}
				if len(got.lines) != len(want.lines) {
					t.Errorf("callback saw %d lines with the counters offered, %d without", len(got.lines), len(want.lines))
				}
				for i := range got.lines {
					if i < len(want.lines) && got.lines[i] != want.lines[i] {
						t.Errorf("callback line %d reported as %d with the counters offered, %d without", i+1, got.lines[i], want.lines[i])
					}
				}
			},
		},
		{
			name:    "duplicates",
			prepare: func(job *FileJob) { Duplicates = true },
			restore: func() { Duplicates = false },
			extra: func(t *testing.T, offered, declined FileJob) {
				// The digest the duplicate check runs on used to be built
				// inside codeState, a byte at a time, from the bytes the
				// counter happened to look at, so it was a property of the
				// counting path and this asserted the two paths agreed on it.
				// It is taken over the whole of the file now, in processFile
				// and after the counting is done, so no counting path can
				// reach it and there is nothing here for the two to disagree
				// about. What is left to check is what the harness checks for
				// every case: the counts themselves.
				if offered.Lines != declined.Lines || offered.Code != declined.Code {
					t.Errorf("counts differ with duplicates on: %d/%d lines, %d/%d code",
						offered.Lines, declined.Lines, offered.Code, declined.Code)
				}
			},
		},
		{
			name:    "cognitive complexity",
			prepare: func(job *FileJob) { Cognitive = true },
			restore: func() { Cognitive = false },
			extra: func(t *testing.T, offered, declined FileJob) {
				if offered.Cognitive == 0 {
					t.Error("no cognitive complexity was counted")
				}
				if offered.Cognitive != declined.Cognitive {
					t.Errorf("cognitive complexity %d with the counters offered, %d without", offered.Cognitive, declined.Cognitive)
				}
			},
		},
		{
			name:    "trace",
			prepare: func(job *FileJob) { Trace = true },
			restore: func() { Trace = false },
			// --trace logs what every line was counted as, and no counter emits
			// those. Counts alone would not notice a counter running here, so
			// the log itself is what is compared.
			capture: true,
			extra: func(t *testing.T, offered, declined FileJob) {
				if !bytes.Contains([]byte(traceOffered), []byte("counted as")) {
					t.Error("no per line trace was written")
				}
				if traceOffered != traceDeclined {
					t.Errorf("trace differs\n  offered : %q\n  declined: %q", traceOffered, traceDeclined)
				}
			},
		},
		{
			name:    "no large",
			prepare: func(job *FileJob) { NoLarge = true; LargeLineCount = 3 },
			restore: func() { NoLarge = false; LargeLineCount = 0 },
		},
		{
			name:    "classify content",
			prepare: func(job *FileJob) { job.ClassifyContent = true },
			extra: func(t *testing.T, offered, declined FileJob) {
				if len(offered.ContentByteType) == 0 {
					t.Error("no byte types were recorded")
				}
				if string(offered.ContentByteType) != string(declined.ContentByteType) {
					t.Error("byte types differ with the counters offered")
				}
			},
		},
		{
			name:    "track complexity lines",
			prepare: func(job *FileJob) { job.TrackComplexityLines = true },
			extra: func(t *testing.T, offered, declined FileJob) {
				var total int64
				for _, n := range offered.ComplexityLine {
					total += n
				}
				if total == 0 {
					t.Error("no per line complexity was recorded")
				}
				if len(offered.ComplexityLine) != len(declined.ComplexityLine) {
					t.Fatalf("complexity lines %d with the counters offered, %d without",
						len(offered.ComplexityLine), len(declined.ComplexityLine))
				}
				for i := range offered.ComplexityLine {
					if offered.ComplexityLine[i] != declined.ComplexityLine[i] {
						t.Errorf("complexity on line %d is %d with the counters offered, %d without",
							i+1, offered.ComplexityLine[i], declined.ComplexityLine[i])
					}
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.restore != nil {
				t.Cleanup(test.restore)
			}

			if test.capture {
				traceOffered, traceDeclined = "", ""
			}

			offered, declined := countWithGuard(t, test.prepare)
			compareCounts(t, "Java", test.name, offered, declined)

			if test.extra != nil {
				test.extra(t, offered, declined)
			}
		})
	}
}

// The guard is a single function now rather than the same seven conditions
// written out twice, so this pins that it answers no to each of them one at a
// time and yes when none of them is set.
func TestSpecialisedCounterEligible(t *testing.T) {
	previous := SpecialisedCounters
	t.Cleanup(func() { SpecialisedCounters = previous })
	SpecialisedCounters = true

	plain := func() *FileJob { return &FileJob{Language: "Java"} }

	if !specialisedCounterEligible(plain()) {
		t.Error("a file asking for nothing extra was declined")
	}

	SpecialisedCounters = false
	if specialisedCounterEligible(plain()) {
		t.Error("a counter was offered with the flag off")
	}
	SpecialisedCounters = true

	for _, test := range []struct {
		name    string
		prepare func(*FileJob)
		restore func()
	}{
		{"duplicates", func(*FileJob) { Duplicates = true }, func() { Duplicates = false }},
		{"cognitive", func(*FileJob) { Cognitive = true }, func() { Cognitive = false }},
		{"trace", func(*FileJob) { Trace = true }, func() { Trace = false }},
		{"no large", func(*FileJob) { NoLarge = true }, func() { NoLarge = false }},
		{"classify content", func(job *FileJob) { job.ClassifyContent = true }, nil},
		{"track complexity lines", func(job *FileJob) { job.TrackComplexityLines = true }, nil},
		{"callback", func(job *FileJob) { job.Callback = &countingCallback{} }, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.restore != nil {
				t.Cleanup(test.restore)
			}

			job := plain()
			test.prepare(job)
			if specialisedCounterEligible(job) {
				t.Errorf("a counter was offered a file asking for %s", test.name)
			}
		})
	}
}
