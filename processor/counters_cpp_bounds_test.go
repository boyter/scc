package processor

import "testing"

// The delimiter of a raw string is read out of the file, so it is the one piece
// of a C++ file that chooses how far the scan looks. These are the shapes where
// that read can reach the end.
func TestCppRawStringBounds(t *testing.T) {
	ProcessConstants()
	previous := SpecialisedCounters
	t.Cleanup(func() { SpecialisedCounters = previous })
	SpecialisedCounters = true

	long := ""
	for i := 0; i < 40; i++ {
		long += "a"
	}

	for _, content := range []string{
		`R"`, `R"(`, `R"()`, `R"()"`, `R"` + long, `R"` + long + `(`,
		`R"` + long + `(x)` + long + `"`, `u8R"`, `uR"`, `UR"`, `LR"`, `u8R`,
		`R"\`, "R\"\n", `R" `, `R"	`, `R")`, `R"()"R"()"`,
		`"R"`, `\R"(x)"`, `R"(`, "R\"(\n", "\xEF\xBB\xBFR\"(x)\"",
	} {
		for _, language := range []string{"C++", "C++ Header"} {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panic counting %s %q: %v", language, content, r)
					}
				}()

				fast := FileJob{Language: language}
				fast.SetContent(content)
				SpecialisedCounters = true
				CountStats(&fast)

				generic := FileJob{Language: language}
				generic.SetContent(content)
				SpecialisedCounters = false
				CountStats(&generic)
				SpecialisedCounters = true

				if countsDiffer(fast, generic) {
					t.Errorf("%s disagrees on %q: counter %d/%d/%d/%d/%d generic %d/%d/%d/%d/%d",
						language, content,
						fast.Lines, fast.Code, fast.Comment, fast.Blank, fast.Complexity,
						generic.Lines, generic.Code, generic.Comment, generic.Blank, generic.Complexity)
				}
			}()
		}
	}
}
