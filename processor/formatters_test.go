package processor

import (
	"strings"
	"testing"
)

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

func TestGetFormattedTime(t *testing.T) {
	res := getFormattedTime()

	if !strings.Contains(res, "T") {
		t.Error("String does not contain expected T", res)
	}

	if !strings.Contains(res, "Z") {
		t.Error("String does not contain expected Z", res)
	}
}

func TestSortSummaryFilesEmpty(t *testing.T) {
	summary := LanguageSummary{}
	sortSummaryFiles(&summary)
}

func TestSortSummaryFiles(t *testing.T) {
	files := []*FileJob{}
	files = append(files, &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	})
	files = append(files, &FileJob{
		Language:           "Go",
		Filename:           "aaaa.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              2000,
		Lines:              2000,
		Code:               2000,
		Comment:            2000,
		Blank:              2000,
		Complexity:         2000,
		WeightedComplexity: 2000,
		Binary:             false,
	})

	summary := LanguageSummary{
		Name:               "Go",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		Count:              1000,
		WeightedComplexity: 1000,
		Files:              files,
	}

	lineSort := []string{"name", "names", "language", "languages", "line", "lines", "RANDOMTHING"}
	for _, val := range lineSort {
		SortBy = val
		sortSummaryFiles(&summary)

		if summary.Files[0].Filename != "aaaa.go" {
			t.Error("Sorting on lines failed", val)
		}
	}

	blankSort := []string{"blank", "blanks"}
	for _, val := range blankSort {
		SortBy = val
		sortSummaryFiles(&summary)

		if summary.Files[0].Filename != "aaaa.go" {
			t.Error("Sorting on blank failed", val)
		}
	}

	codeSort := []string{"code", "codes"}
	for _, val := range codeSort {
		SortBy = val
		sortSummaryFiles(&summary)

		if summary.Files[0].Filename != "aaaa.go" {
			t.Error("Sorting on code failed", val)
		}
	}

	commentSort := []string{"comment", "comments"}
	for _, val := range commentSort {
		SortBy = val
		sortSummaryFiles(&summary)

		if summary.Files[0].Filename != "aaaa.go" {
			t.Error("Sorting on comment failed", val)
		}
	}

	complexitySort := []string{"complexity", "complexitys"}
	for _, val := range complexitySort {
		SortBy = val
		sortSummaryFiles(&summary)

		if summary.Files[0].Filename != "aaaa.go" {
			t.Error("Sorting on complexity failed", val)
		}
	}
}

func TestToJSONEmpty(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	close(inputChan)
	res := toJSON(inputChan)

	if res != "[]" {
		t.Error("Expected empty JSON return", res)
	}
}

func TestToJSONSingle(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}
	close(inputChan)
	Debug = true // Increase coverage slightly
	res := toJSON(inputChan)
	Debug = false

	if !strings.Contains(res, `"Name":"Go"`) || !strings.Contains(res, `"Code":1000`) {
		t.Error("Expected JSON return", res)
	}
}

func TestToJSONMultiple(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "aaaa.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}
	close(inputChan)
	Debug = true // Increase coverage slightly
	res := toJSON(inputChan)
	Debug = false

	if !strings.Contains(res, `aaaa.go`) || !strings.Contains(res, `bbbb.go`) {
		t.Error("Expected JSON return", res)
	}
}

func TestToCsvMultiple(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "aaaa.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}
	close(inputChan)
	Debug = true // Increase coverage slightly
	res := toCSV(inputChan)
	Debug = false

	if !strings.Contains(res, `aaaa.go,`) || !strings.Contains(res, `bbbb.go`) {
		t.Error("Expected CSV return", res)
	}
}

func TestFileSummarizeWide(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}

	close(inputChan)
	Format = "wide"
	More = true
	res := fileSummarize(inputChan)
	More = false

	if !strings.Contains(res, `Language`) {
		t.Error("Expected CSV return", res)
	}
}

func TestFileSummarizeJson(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}

	close(inputChan)
	Format = "JSON"
	More = false
	res := fileSummarize(inputChan)

	if !strings.Contains(res, `bbbb.go`) || !strings.HasPrefix(res, "[") {
		t.Error("Expected JSON return", res)
	}
}

func TestFileSummarizeCsv(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}

	close(inputChan)
	Format = "CSV"
	More = false
	res := fileSummarize(inputChan)

	if !strings.Contains(res, `bbbb.go`) {
		t.Error("Expected CSV return", res)
	}
}

func TestFileSummarizeDefault(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              1000,
		Lines:              1000,
		Code:               1000,
		Comment:            1000,
		Blank:              1000,
		Complexity:         1000,
		WeightedComplexity: 1000,
		Binary:             false,
	}

	close(inputChan)
	Format = ""
	More = false
	res := fileSummarize(inputChan)

	if !strings.Contains(res, `Estimated Cost to Develop`) {
		t.Error("Expected summary return", res)
	}
}

// When using columise  ~28726 ns/op
// When using optimised ~14293 ns/op
func BenchmarkFileSummerize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		fileSummaryJobQueue := make(chan *FileJob, 1000)

		fileSummaryJobQueue <- &FileJob{
			Blank:      1,
			Bytes:      1,
			Code:       1,
			Comment:    1,
			Complexity: 1,
			Language:   "Go",
			Lines:      10,
		}
		close(fileSummaryJobQueue)
		b.StartTimer()

		fileSummarize(fileSummaryJobQueue)
	}
}
