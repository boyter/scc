package processor

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestIsWhitespace(t *testing.T) {
	if !isWhitespace(' ') {
		t.Errorf("Expected to be true")
	}
}

func TestCountStatsLines(t *testing.T) {
	Trace = false
	Debug = false
	Verbose = false

	fileJob := FileJob{
		Content: []byte(""),
		Lines:   0,
	}

	// Both tokei and sloccount count this as 0 so lets follow suit
	// cloc ignores the file itself because it is empty
	CountStats(&fileJob)
	if fileJob.Lines != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Lines)
	}

	// Interestingly this file would be 0 lines in "wc -l" because it only counts newlines
	// all others count this as 1
	fileJob.Lines = 0
	fileJob.Content = []byte("a")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.Content = []byte("a\n")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	// tokei counts this as 1 because its still on a single line unless something follows
	// the newline its still 1 line
	fileJob.Lines = 0
	fileJob.Content = []byte("1\n")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.Content = []byte("1\n2\n")
	CountStats(&fileJob)
	if fileJob.Lines != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.Content = []byte("1\n2\n3")
	CountStats(&fileJob)
	if fileJob.Lines != 3 {
		t.Errorf("Three lines expected got %d", fileJob.Lines)
	}

	content := ""
	for i := 0; i < 5000; i++ {
		content += "a\n"
		fileJob.Lines = 0
		fileJob.Content = []byte(content)
		CountStats(&fileJob)
		if fileJob.Lines != int64(i+1) {
			t.Errorf("Expected %d got %d", i+1, fileJob.Lines)
		}
	}
}

func TestCountStatsCode(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
		Code:    0,
	}

	// Both tokei and sloccount count this as 0 so lets follow suit
	// cloc ignores the file itself because it is empty
	CountStats(&fileJob)
	if fileJob.Code != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Code)
	}

	// Interestingly this file would be 0 lines in "wc -l" because it only counts newlines
	// all others count this as 1
	fileJob.Code = 0
	fileJob.Content = []byte("a")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("i++ # comment")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("i++ // comment")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("a\n")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	// tokei counts this as 1 because its still on a single line unless something follows
	// the newline its still 1 line
	fileJob.Code = 0
	fileJob.Content = []byte("1\n")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("1\n2\n")
	CountStats(&fileJob)
	if fileJob.Code != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.Content = []byte("1\n2\n3")
	CountStats(&fileJob)
	if fileJob.Code != 3 {
		t.Errorf("Three lines expected got %d", fileJob.Code)
	}

	content := ""
	for i := 0; i < 100; i++ {
		content += "a\n"
		fileJob.Code = 0
		fileJob.Content = []byte(content)
		CountStats(&fileJob)
		if fileJob.Code != int64(i+1) {
			t.Errorf("Expected %d got %d", i+1, fileJob.Code)
		}
	}
}

func TestCountStatsWithQuotes(t *testing.T) {
	fileJob := FileJob{}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.Content = []byte(`var test = "/*";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.Content = []byte(`t = " if ";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("No line expected got %d", fileJob.Complexity)
	}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.Content = []byte(`t = " if switch for while do loop != == && || ";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("No line expected got %d", fileJob.Complexity)
	}
}

func TestCountStatsBlankLines(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
		Blank:   0,
	}

	CountStats(&fileJob)
	if fileJob.Blank != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte(" ")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\n")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\n ")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("            ")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("            \n             ")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\r\n\r\n")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.Content = []byte("\r\n")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}
}

func TestCountStatsComplexityCount(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	checks := []string{
		"if ",
		"	if ",
		"if a.equals(b) {",
		"if(",
		" if(i.equals(0))",
		"    if(",
		"    if( ",
	}

	for _, check := range checks {
		fileJob.Complexity = 0
		fileJob.Content = []byte(check)
		fileJob.Language = "Java"
		CountStats(&fileJob)
		if fileJob.Complexity != 1 {
			t.Errorf("Expected complexity of 1 got %d for %s", fileJob.Complexity, check)
		}
	}
}

func TestCountStatsComplexityCountFalse(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	checks := []string{
		"if",
		"aif ",
		"aif(",
	}

	for _, check := range checks {
		fileJob.Complexity = 0
		fileJob.Content = []byte(check)
		fileJob.Language = "Java"
		CountStats(&fileJob)
		if fileJob.Complexity != 0 {
			t.Errorf("Expected complexity of 0 got %d for %s", fileJob.Complexity, check)
		}
	}

}

type linecounter struct {
	blanks   int
	comments int
	code     int
	loc      int
	stop     bool
}

func (l *linecounter) ProcessLine(job *FileJob, currentLine int64, lineType LineType) bool {
	l.loc++
	switch lineType {
	case LINE_BLANK:
		l.blanks++
	case LINE_COMMENT:
		l.comments++
	case LINE_CODE:
		l.code++
	}
	return !l.stop
}

func TestCountStatsCallback(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	fileJob.Content = []byte(`package foo

import com.foo.bar;

// this is a comment
class A {
}`)
	var lc linecounter
	fileJob.Language = "Java"
	fileJob.Callback = &lc
	CountStats(&fileJob)
	if lc.loc != 7 {
		t.Errorf("Expected loc of 7 got %d", lc.loc)
	}
	if lc.blanks != 2 {
		t.Errorf("Expected loc of 2 got %d", lc.blanks)
	}
	if lc.comments != 1 {
		t.Errorf("Expected loc of 1 got %d", lc.comments)
	}
	if lc.code != 4 {
		t.Errorf("Expected loc of 4 got %d", lc.code)
	}
}

func TestCountStatsCallbackInterrupt(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	fileJob.Content = []byte(`package foo

import com.foo.bar;

// this is a comment
class A {
}`)
	var lc linecounter
	lc.stop = true
	fileJob.Language = "Java"
	fileJob.Callback = &lc
	CountStats(&fileJob)
	if lc.loc != 1 {
		t.Errorf("Expected loc of 1 got %d", lc.loc)
	}
}

// Edge case condition where if ending with comment it would be counted
// as code due to how internal state work.
func TestCountStatsEdgeCase1(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.Content = []byte(`/**/
`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}
}

// Turns out that some languages such as Rust support
// nested comments. Check that it works here
func TestCountStatsNestedComments(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.Content = []byte(`/*/**/*/`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

// Java does not support nested multiline comments
func TestCountStatsNestedCommentsJava(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.Content = []byte(`/*/**/*/`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsNestedCommentsRegression(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.Content = []byte(`t/*/**/*/`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsSingleCommentRegression(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.Content = []byte(`t = "
/*
";`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsStringCheck(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.Content = []byte(`let does_not_start = // "
"until here,
test/*
test"; // a quote: "`)

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comment lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blank lines got %d", fileJob.Blank)
	}
}

func TestCheckForMatchNoMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("one does not simply walk into mordor"),
	}

	matches := &Trie{}
	matches.Insert(TSlcomment, []byte("//"))
	matches.Insert(TSlcomment, []byte("--"))

	match, _, _ := matches.Match(fileJob.Content)

	if match != 0 {
		t.Errorf("Expected no match")
	}
}

func TestCheckForMatchHasMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("// one does not simply walk into mordor"),
	}

	matches := &Trie{}
	matches.Insert(TSlcomment, []byte("//"))
	matches.Insert(TSlcomment, []byte("--"))

	match, _, _ := matches.Match(fileJob.Content)

	if match != TSlcomment {
		t.Errorf("Expected match")
	}
}

func TestCheckForMatchSingleNoMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("// one does not simply walk into mordor"),
	}

	matches := []byte("*/")

	match := checkForMatchSingle('/', 0, 100, matches, &fileJob)

	if match != false {
		t.Errorf("Expected no match")
	}
}

func TestCheckForMatchSingleMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("*/ one does not simply walk into mordor"),
	}

	matches := []byte("*/")

	match := checkForMatchSingle('*', 0, 100, matches, &fileJob)

	if match != true {
		t.Errorf("Expected match")
	}
}

func TestCheckComplexityMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Java",
		Content:  []byte("for (int i=0; i<100; i++) {"),
	}

	matches := &Trie{}
	matches.Insert(TComplexity, []byte("for "))
	matches.Insert(TComplexity, []byte("for("))

	match, n, _ := matches.Match(fileJob.Content)

	if match != TComplexity || n != 4 {
		t.Errorf("Expected match")
	}
}

func TestCheckComplexityNoMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Java",
		Content:  []byte("far (int i=0; i<100; i++) {"),
	}

	matches := &Trie{}
	matches.Insert(TComplexity, []byte("for "))
	matches.Insert(TComplexity, []byte("for("))

	match, _, _ := matches.Match(fileJob.Content)

	if match != 0 {
		t.Errorf("Expected no match")
	}
}

func TestCountStatsRubyRegression(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Ruby",
	}

	fileJob.Content = []byte(`=begin
=end
t`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comment lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blank lines got %d", fileJob.Blank)
	}
}

func TestFileProcessorWorker(t *testing.T) {
	inputChan := make(chan *FileJob, 10000)

	inputChan <- &FileJob{
		Filename:  "testing.go",
		Location:  "./",
		Extension: "go",
		Content:   []byte("this is some content"),
	}

	close(inputChan)
	outputChan := make(chan *FileJob, 10000)

	Duplicates = true

	fileProcessorWorker(inputChan, outputChan)

	for res := range outputChan {
		if res.Bytes == 0 {
			t.Error("Expect bytes to have something")
		}
	}
}

func TestGuessLanguageCoq(t *testing.T) {
	fileJob := &FileJob{
		PossibleLanguages: []string{"Coq", "SystemVerilog"},
		Content:           []byte(`Require Hypothesis Inductive`),
	}

	determineLanguage(fileJob)

	if fileJob.Language != "Coq" {
		t.Error("Expected guessed language to have been Coq got", fileJob.Language)
	}
}

func TestGuessLanguageSystemVerilog(t *testing.T) {
	fileJob := &FileJob{
		PossibleLanguages: []string{"Coq", "SystemVerilog"},
		Content:           []byte(`endmodule posedge edge always wire`),
	}

	determineLanguage(fileJob)

	if fileJob.Language != "SystemVerilog" {
		t.Error("Expected guessed language to have been SystemVerilog got", fileJob.Language)
	}
}

func TestGuessLanguageLanguageSetNoPossible(t *testing.T) {
	fileJob := &FileJob{
		Language: "Java",
		Content:  []byte(`endmodule posedge edge always wire`),
	}

	determineLanguage(fileJob)

	if fileJob.Language != "Java" {
		t.Error("Expected guessed language to have been Java got", fileJob.Language)
	}
}

func TestGuessLanguageSingleLanguageSet(t *testing.T) {
	fileJob := &FileJob{
		Language:          "Java",
		PossibleLanguages: []string{"Rust"},
		Content:           []byte(`endmodule posedge edge always wire`),
	}

	determineLanguage(fileJob)

	if fileJob.Language != "Rust" {
		t.Error("Expected guessed language to have been Rust got", fileJob.Language)
	}
}

func TestGuessLanguageLanguageEmptyContent(t *testing.T) {
	fileJob := &FileJob{
		PossibleLanguages: []string{"Rust"},
		Content:           []byte(``),
	}

	determineLanguage(fileJob)

	if fileJob.Language != "Rust" {
		t.Error("Expected guessed language to have been Rust got", fileJob.Language)
	}
}

//////////////////////////////////////////////////
// Benchmarks Below
//////////////////////////////////////////////////

func BenchmarkCountStatsLinesEmpty(b *testing.B) {
	fileJob := FileJob{
		Content: []byte(""),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesSingleChar(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesTwoLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a\na"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a\na\na"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesShortLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesShortEmptyLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeShortLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890\n1234567890\n1234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeShortEmptyLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("          \n          \n          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongMixedLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890          1234567890          1234567890          1234567890          1234567890          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongAlternateLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLines(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 500; i++ {
		content += "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLinesTriggerComplexityIf(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 500; i++ {
		content += "iiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiii\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLinesTriggerComplexityFor(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 500; i++ {
		content += "fofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofo\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFourHundredLongLinesMixed(b *testing.B) {
	b.StopTimer()
	content := ""
	for i := 0; i < 100; i++ {
		content += "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n"
		content += "1234567890          1234567890          1234567890          1234567890          1234567890          \n"
		content += "                                                                                                    \n"
		content += "a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a \n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCheckByteEqualityReflect(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := reflect.DeepEqual(one[1:], two[1:])

		if equal {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckByteEqualityBytes(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := bytes.Equal(one[1:], two[1:])

		if equal {
			count++
		}
	}

	b.Log(count)
}

// This appears to be faster than bytes.Equal because it does not need
// to do length comparison checks at the start
func BenchmarkCheckByteEqualityLoop(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := true

		for j := 1; j < len(one); j++ {
			if one[j] != two[j] {
				equal = false
				break
			}
		}

		if equal {
			count++
		}
	}

	b.Log(count)
}

// Check if the 1 offset makes a difference, which it does by ~1 ns
func BenchmarkCheckByteEqualityLoopWithAddtional(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := true

		// Don't start at 1 like the above but 0 to do a full scan
		for j := 0; j < len(one); j++ {
			if one[j] != two[j] {
				equal = false
				break
			}
		}

		if equal {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckArrayCheck(b *testing.B) {
	array := []byte{
		'a',
		'b',
		'c',
		'd',
		'e',
		'f',
		'g',
		'h',
		'i',
		'j',
	}

	var searchFor byte = 'j'
	found := 0

	for i := 0; i < b.N; i++ {
		for index := 0; index < len(array); index++ {
			if array[index] == searchFor {
				found++
				break
			}
		}
	}

	b.Log(found)
}

func BenchmarkCheckMapCheck(b *testing.B) {
	array := map[byte]bool{
		'a': true,
		'b': true,
		'c': true,
		'd': true,
		'e': true,
		'f': true,
		'g': true,
		'h': true,
		'i': true,
		'j': true,
	}

	var searchFor byte = 'j'
	found := 0

	for i := 0; i < b.N; i++ {

		_, ok := array[searchFor]

		if ok {
			found++
		}
	}

	b.Log(found)
}

func BenchmarkStringLoop(b *testing.B) {
	b.StopTimer()

	var str strings.Builder
	for i := 0; i < 10000; i++ {
		str.WriteString("1")
	}
	search := str.String()
	count := 0
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < len(search); j++ {
			if search[j] != '\n' {
				count++
			}

		}
	}
	b.Log(count)
}

func BenchmarkByteLoop(b *testing.B) {
	b.StopTimer()

	var str strings.Builder
	for i := 0; i < 10000; i++ {
		str.WriteString("1")
	}
	search := []byte(str.String())
	count := 0
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < len(search); j++ {
			if search[j] != '\n' {
				count++
			}

		}
	}
	b.Log(count)
}

func BenchmarkLoopInLoop(b *testing.B) {
	search := []byte("this is a long from for string which we will search")
	matches := [][]byte{
		[]byte("if"),
		[]byte("if("),
		[]byte("else"),
		[]byte("while"),
		[]byte("while("),
		[]byte("for"),
		[]byte("foreach"),
	}
	endPoint := len(search)
	b.ResetTimer()

	potentialMatch := true
	for i := 0; i < b.N; i++ {

		potentialMatch = true
		for index := 0; index < len(search); index++ {

			for k := 0; k < len(matches); k++ {

				for j := 0; j < len(matches[k]); j++ {
					if index+j >= endPoint || matches[k][j] != search[index+j] {
						potentialMatch = false
					}
				}
			}

		}

	}
	b.Log(potentialMatch)
}

func BenchmarkFlattenedLoop(b *testing.B) {
	index := 0
	search := []byte("this is a long from for string which we will search")
	matches := []byte("if if( else while while( for foreach")

	b.ResetTimer()

	potentialMatch := true
	count := 0
	for i := 0; i < b.N; i++ {

		potentialMatch = true
		for j := 0; j < len(matches); j++ {
			if matches[j] == ' ' {
				count = 0
			} else {
				if matches[j] != search[index+count] {
					potentialMatch = false
				}

			}
		}

	}

	b.Log(potentialMatch)
}

func BenchmarkCheckComplexity(b *testing.B) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Java",
		Content:  []byte("A little while ago, I passed my first year mark of working for Google. This also marked the "),
	}

	matches := &Trie{}
	matches.Insert(TComplexity, []byte("for "))
	matches.Insert(TComplexity, []byte("for("))
	matches.Insert(TComplexity, []byte("if "))
	matches.Insert(TComplexity, []byte("if("))
	matches.Insert(TComplexity, []byte("switch "))
	matches.Insert(TComplexity, []byte("while "))
	matches.Insert(TComplexity, []byte("else "))
	matches.Insert(TComplexity, []byte("|| "))
	matches.Insert(TComplexity, []byte("&& "))
	matches.Insert(TComplexity, []byte("!= "))
	matches.Insert(TComplexity, []byte("== "))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(fileJob.Content); j++ {
			matches.Match(fileJob.Content)
		}
	}
}

func BenchmarkCheckLen(b *testing.B) {
	matches := [][]byte{
		[]byte("for "),
		[]byte("for("),
		[]byte("if "),
		[]byte("if("),
		[]byte("switch "),
		[]byte("while "),
		[]byte("else "),
		[]byte("|| "),
		[]byte("&& "),
		[]byte("!= "),
		[]byte("== "),
	}

	count := 0
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(matches); j++ {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckLenPrecalc(b *testing.B) {
	matches := [][]byte{
		[]byte("for "),
		[]byte("for("),
		[]byte("if "),
		[]byte("if("),
		[]byte("switch "),
		[]byte("while "),
		[]byte("else "),
		[]byte("|| "),
		[]byte("&& "),
		[]byte("!= "),
		[]byte("== "),
	}

	count := 0
	for i := 0; i < b.N; i++ {
		l := len(matches)
		for j := 0; j < l; j++ {
			count++
		}
	}

	b.Log(count)
}
