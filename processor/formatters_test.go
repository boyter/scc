// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
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

func TestPrintError(t *testing.T) {
	printError("Testing print error")
}

func TestPrintWarnF(t *testing.T) {
	printWarnf("Testing print error")
}

func TestPrintDebugF(t *testing.T) {
	printDebugf("Testing print error")
}

func TestPrintTraceF(t *testing.T) {
	printTracef("Testing print error")
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

func TestCalculateCocomo(t *testing.T) {
	var str strings.Builder
	calculateCocomo(1, &str)

	if !strings.Contains(str.String(), "Estimated Schedule Effort (organic) 0.22 months") {
		t.Error("expected to match got", str.String())
	}
}

func TestCalculateSizeSingleByte(t *testing.T) {
	var str strings.Builder
	calculateSize(1, &str)

	if !strings.Contains(str.String(), "Processed 1 bytes, 0.000 megabytes (SI)") {
		t.Error("expected to match got", str.String())
	}
}

func TestCalculateSize(t *testing.T) {
	var str strings.Builder
	calculateSize(1000000, &str)

	if !strings.Contains(str.String(), "Processed 1000000 bytes, 1.000 megabytes (SI)") {
		t.Error("expected to match got", str.String())
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
		Location:           "./bbbb.go",
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
		Location:           "./aaaa.go",
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

func TestSortSummaryFilesName(t *testing.T) {
	goFiles := []*FileJob{}
	goFiles = append(goFiles, &FileJob{
		Language: "Go",
		Location: "bbbb.go",
	})

	goFiles = append(goFiles, &FileJob{
		Language: "Go",
		Location: "aaaa.go",
	})

	goFiles = append(goFiles, &FileJob{
		Language: "Go",
		Location: "cccc.go",
	})

	summary := LanguageSummary{
		Name:  "Go",
		Files: goFiles,
	}

	lineSort := []string{"name", "names", "language", "languages"}
	for _, val := range lineSort {
		SortBy = val
		sortSummaryFiles(&summary)

		if summary.Files[0].Location != "aaaa.go" {
			t.Error("Sorting on lines failed", val)
		}
	}
	SortBy = ""
}

func TestSortLanguageSummaryName(t *testing.T) {
	SortBy = "name"
	ls := []LanguageSummary{
		{
			Name:  "b",
			Lines: 1,
		},
		{
			Name:  "a",
			Lines: 1,
		},
	}

	ls = sortLanguageSummary(ls)

	if ls[0].Name != "a" {
		t.Error("Expected a to be first")
	}
}

func TestSortLanguageSummaryLine(t *testing.T) {
	SortBy = "line"
	ls := []LanguageSummary{
		{
			Name:  "a",
			Lines: 1,
		},
		{
			Name:  "b",
			Lines: 1,
		},
		{
			Name:  "c",
			Lines: 2,
		},
	}

	ls = sortLanguageSummary(ls)

	if ls[0].Name != "c" || ls[1].Name != "a" {
		t.Error("Expected c to be first and a second")
	}
}

func TestSortLanguageSummaryBlank(t *testing.T) {
	SortBy = "blank"
	ls := []LanguageSummary{
		{
			Name:  "a",
			Blank: 1,
		},
		{
			Name:  "b",
			Blank: 1,
		},
		{
			Name:  "c",
			Blank: 2,
		},
	}

	ls = sortLanguageSummary(ls)

	if ls[0].Name != "c" || ls[1].Name != "a" {
		t.Error("Expected c to be first and a second")
	}
}

func TestSortLanguageSummaryCode(t *testing.T) {
	SortBy = "code"
	ls := []LanguageSummary{
		{
			Name: "a",
			Code: 1,
		},
		{
			Name: "b",
			Code: 1,
		},
		{
			Name: "c",
			Code: 2,
		},
	}

	ls = sortLanguageSummary(ls)

	if ls[0].Name != "c" || ls[1].Name != "a" {
		t.Error("Expected c to be first and a second")
	}
}

func TestSortLanguageSummaryComment(t *testing.T) {
	SortBy = "comment"
	ls := []LanguageSummary{
		{
			Name:    "a",
			Comment: 1,
		},
		{
			Name:    "b",
			Comment: 1,
		},
		{
			Name:    "c",
			Comment: 2,
		},
	}

	ls = sortLanguageSummary(ls)

	if ls[0].Name != "c" || ls[1].Name != "a" {
		t.Error("Expected c to be first and a second")
	}
}

func TestSortLanguageSummaryComplexity(t *testing.T) {
	SortBy = "complexity"
	ls := []LanguageSummary{
		{
			Name:       "a",
			Complexity: 1,
		},
		{
			Name:       "b",
			Complexity: 1,
		},
		{
			Name:       "c",
			Complexity: 2,
		},
	}

	ls = sortLanguageSummary(ls)

	if ls[0].Name != "c" || ls[1].Name != "a" {
		t.Error("Expected c to be first and a second")
	}
}

func TestSortSummaryNames(t *testing.T) {
	SortBy = "name"
	ls := []LanguageSummary{
		{
			Name:       "a",
			Complexity: 1,
		},
		{
			Name:       "b",
			Complexity: 1,
		},
		{
			Name:       "c",
			Complexity: 2,
		},
	}

	ls = sortLanguageSummary(ls)

	if ls[0].Name != "a" || ls[1].Name != "b" || ls[2].Name != "c" {
		t.Error("Expected a to be first and b second and c third")
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
	Files = true
	res := toJSON(inputChan)
	Debug = false

	if !strings.Contains(res, `"Name":"Go"`) || !strings.Contains(res, `"Code":1000`) || !strings.Contains(res, `"Filename":"bbbb.go"`) {
		t.Error("Expected JSON return", res)
	}
	if strings.Contains(res, `"Content":`) {
		t.Error("Expected JSON return", res)
	}
}

func TestToJSONSingleWithoutFiles(t *testing.T) {
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
	Files = false
	res := toJSON(inputChan)
	Debug = false

	if !strings.Contains(res, `"Name":"Go"`) || !strings.Contains(res, `"Code":1000`) {
		t.Error("Expected JSON return", res)
	}
	if strings.Contains(res, `"Filename":"bbbb.go"`) {
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
	Files = true
	res := toJSON(inputChan)
	Debug = false

	if !strings.Contains(res, `aaaa.go`) || !strings.Contains(res, `bbbb.go`) {
		t.Error("Expected JSON return", res)
	}
}

func TestToYAMLEmpty(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	close(inputChan)
	res := toClocYAML(inputChan)

	if !strings.Contains(res, "{}") || !strings.Contains(res, "header:") || !strings.Contains(res, "n_files: 0") {
		t.Error("Expected empty Cloc YAML return", res)
	}
}

func TestToYAMLSingle(t *testing.T) {
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
	res := toClocYAML(inputChan)
	Debug = false

	if !strings.Contains(res, `n_lines: 1000`) {
		t.Error("Expected Cloc YAML return", res)
	}
}

func TestToYAMLMultiple(t *testing.T) {
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
	res := toClocYAML(inputChan)
	Debug = false

	if !strings.Contains(res, `code: 2000`) || !strings.Contains(res, `n_lines: 2000`) {
		t.Error("Expected Cloc JSON return", res)
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

func TestToCsvStreamMultiple(t *testing.T) {
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
	res := toCSVStream(inputChan)
	Debug = false

	if res != "" {
		t.Error("Expected CSV return", res)
	}
}

func TestToCsvFilesSorted(t *testing.T) {
	fj1 := &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "./",
		Bytes:              90,
		Lines:              90,
		Code:               90,
		Comment:            90,
		Blank:              90,
		Complexity:         90,
		WeightedComplexity: 90,
		Binary:             false,
	}
	fj2 := &FileJob{
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

	Files = true
	SortBy = "lines"

	inputChan1 := make(chan *FileJob, 1000)
	inputChan1 <- fj1
	inputChan1 <- fj2
	close(inputChan1)
	res1 := toCSV(inputChan1)

	inputChan2 := make(chan *FileJob, 1000)
	inputChan2 <- fj2
	inputChan2 <- fj1
	close(inputChan2)
	res2 := toCSV(inputChan2)

	Files = false

	if res1 != res2 {
		t.Error("Should be sorted to be the same")
	}
}

func TestToOpenMetricsMultiple(t *testing.T) {
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
	Files = false
	Debug = true // Increase coverage slightly
	res := toOpenMetrics(inputChan)
	Debug = false

	var expectedResult = `# TYPE scc_files gauge
# HELP scc_files Number of sourcecode files.
# TYPE scc_lines gauge
# HELP scc_lines Number of lines.
# TYPE scc_code gauge
# HELP scc_code Number of lines of actual code.
# TYPE scc_comments gauge
# HELP scc_comments Number of comments.
# TYPE scc_blanks gauge
# HELP scc_blanks Number of blank lines.
# TYPE scc_complexity gauge
# HELP scc_complexity Code complexity.
# TYPE scc_bytes gauge
# UNIT scc_bytes bytes
# HELP scc_bytes Size in bytes.
scc_files{language="Go"} 2
scc_lines{language="Go"} 2000
scc_code{language="Go"} 2000
scc_comments{language="Go"} 2000
scc_blanks{language="Go"} 2000
scc_complexity{language="Go"} 2000
scc_bytes{language="Go"} 2000
`

	if res != expectedResult {
		t.Error("Expected OpenMetrics return", res)
	}
}

func TestToSQLSingle(t *testing.T) {
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
		Uloc:               99,
	}
	close(inputChan)
	Files = false
	Debug = true // Increase coverage slightly
	res := toSql(inputChan)
	Debug = false

	if !strings.Contains(res, `create table metadata`) {
		t.Error("Expected create table return", res)
	}

	if !strings.Contains(res, `create table t`) {
		t.Error("Expected create table return", res)
	}

	if !strings.Contains(res, `begin transaction`) {
		t.Error("Expected begin transaction return", res)
	}

	if !strings.Contains(res, `insert into t values('', 'Go', './', './', 'bbbb.go', 1000, 1000, 1000, 1000, 1000, 99);`) {
		t.Error("Expected insert return", res)
	}

	if !strings.Contains(res, `insert into metadata values`) {
		t.Error("Expected insert return", res)
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
	Files = true
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

func TestFileSummarizeYaml(t *testing.T) {
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
	Format = "cloc-yml"
	More = false
	res := fileSummarize(inputChan)

	if !strings.Contains(res, `code: 1000`) {
		t.Error("Expected YAML return", res)
	}
}

func TestFileSummarizeYml(t *testing.T) {
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
	Format = "cloc-YAML"
	More = false
	res := fileSummarize(inputChan)

	if !strings.Contains(res, `code: 1000`) {
		t.Error("Expected YML return", res)
	}
}

func TestFileSummarizeOpenMetrics(t *testing.T) {
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
	Files = false
	Format = "OpenMetrics"
	More = false
	res := fileSummarize(inputChan)

	var expectedResult = `# TYPE scc_files gauge
# HELP scc_files Number of sourcecode files.
# TYPE scc_lines gauge
# HELP scc_lines Number of lines.
# TYPE scc_code gauge
# HELP scc_code Number of lines of actual code.
# TYPE scc_comments gauge
# HELP scc_comments Number of comments.
# TYPE scc_blanks gauge
# HELP scc_blanks Number of blank lines.
# TYPE scc_complexity gauge
# HELP scc_complexity Code complexity.
# TYPE scc_bytes gauge
# UNIT scc_bytes bytes
# HELP scc_bytes Size in bytes.
scc_files{language="Go"} 1
scc_lines{language="Go"} 1000
scc_code{language="Go"} 1000
scc_comments{language="Go"} 1000
scc_blanks{language="Go"} 1000
scc_complexity{language="Go"} 1000
scc_bytes{language="Go"} 1000
`

	if res != expectedResult {
		t.Error("Expected OpenMetrics return", res)
	}
}

func TestFileSummarizeOpenMetricsPerFile(t *testing.T) {
	inputChan := make(chan *FileJob, 1000)
	inputChan <- &FileJob{
		Language:           "Go",
		Filename:           "bbbb.go",
		Extension:          "go",
		Location:           "C:\\bbbb.go", // to test escaping of the backslash
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
	Format = "OpenMetrics"
	More = false
	Files = true
	res := fileSummarize(inputChan)

	var expectedResult = `# TYPE scc_files gauge
# HELP scc_files Number of sourcecode files.
# TYPE scc_lines gauge
# HELP scc_lines Number of lines.
# TYPE scc_code gauge
# HELP scc_code Number of lines of actual code.
# TYPE scc_comments gauge
# HELP scc_comments Number of comments.
# TYPE scc_blanks gauge
# HELP scc_blanks Number of blank lines.
# TYPE scc_complexity gauge
# HELP scc_complexity Code complexity.
# TYPE scc_bytes gauge
# UNIT scc_bytes bytes
# HELP scc_bytes Size in bytes.
scc_lines{language="Go",file="C:\\bbbb.go"} 1000
scc_code{language="Go",file="C:\\bbbb.go"} 1000
scc_comments{language="Go",file="C:\\bbbb.go"} 1000
scc_blanks{language="Go",file="C:\\bbbb.go"} 1000
scc_complexity{language="Go",file="C:\\bbbb.go"} 1000
scc_bytes{language="Go",file="C:\\bbbb.go"} 1000
# EOF
`

	if res != expectedResult {
		t.Error("Expected OpenMetrics return", res)
	}
}

func TestFileSummarizeHtml(t *testing.T) {
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
	Format = "html"
	More = false
	res := fileSummarize(inputChan)

	if !strings.Contains(res, `<th>1000`) {
		t.Error("Expected HTML return", res)
	}
}

func TestFileSummarizeHtmlTable(t *testing.T) {
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
	Format = "html-table"
	More = false
	res := fileSummarize(inputChan)

	if !strings.Contains(res, `<th>1000`) {
		t.Error("Expected HTML-table return", res)
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

func TestFileSummarizeLong(t *testing.T) {
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
	res := fileSummarizeLong(inputChan)

	if !strings.Contains(res, `Language`) {
		t.Error("Expected Summary return", res)
	}
}

func TestFileSummarizeShort(t *testing.T) {
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
	res := fileSummarizeShort(inputChan)

	if !strings.Contains(res, `Language`) {
		t.Error("Expected Summary return", res)
	}
}

func TestFileSummarizeShortSort(t *testing.T) {
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

	sortBy := []string{"name", "line", "blank", "code", "comment"}

	Files = true
	for _, sort := range sortBy {
		SortBy = sort
		res := fileSummarizeShort(inputChan)

		if !strings.Contains(res, `Language`) {
			t.Error("Expected Summary return", res)
		}
	}
}

func TestFileSummarizeLongSort(t *testing.T) {
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

	sortBy := []string{"name", "line", "blank", "code", "comment"}

	Files = true
	for _, sort := range sortBy {
		SortBy = sort
		res := fileSummarizeLong(inputChan)

		if !strings.Contains(res, `Language`) {
			t.Error("Expected Summary return", res)
		}
	}
}

func TestGetTabularShortBreak(t *testing.T) {
	Ci = false
	r := getTabularShortBreak()

	if !strings.Contains(r, "─") {
		t.Errorf("Expected to have box line")
	}

	Ci = true
	r = getTabularShortBreak()

	if !strings.Contains(r, "-") {
		t.Errorf("Expected to have hyphen")
	}

	Ci = false
}

func TestGetTabularWideBreak(t *testing.T) {
	{
		Ci, HBorder = false, false
		r := getTabularWideBreak()
		if !strings.Contains(r, "─") {
			t.Errorf("Expected to have box line")
		}
	}
	{
		Ci, HBorder = false, true
		r := getTabularWideBreak()
		if strings.Contains(r, "─") {
			t.Errorf("Didn't expect to have box line")
		}
	}
	{
		Ci, HBorder = true, false
		r := getTabularWideBreak()
		if !strings.Contains(r, "-") {
			t.Errorf("Expected to have hyphen")
		}
	}
	{
		Ci, HBorder = true, true
		r := getTabularWideBreak()
		if strings.Contains(r, "-") {
			t.Errorf("Didn't expect to have hyphen")
		}
	}

	Ci, HBorder = false, false
}

func TestToHTML(t *testing.T) {
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
	res := toHtml(inputChan)

	if !strings.Contains(res, `<html lang="en">`) {
		t.Error("Expected to have HTML wrapper")
	}
}

func TestToHTMLTable(t *testing.T) {
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
	res := toHtmlTable(inputChan)

	if strings.Contains(res, `<html lang="en">`) {
		t.Error("Expected to not have wrapper")
	}

	if !strings.Contains(res, `<table id="scc-table">`) {
		t.Error("Expected to have table element")
	}
}

func TestUnicodeAwareTrimAscii(t *testing.T) {
	tmp := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.md"
	res := unicodeAwareTrim(tmp, shortFormatFileTruncate)
	if res != "~aaaaaaaaaaaaaaaaaaaaaaaaaa.md" {
		t.Error("expected ~aaaaaaaaaaaaaaaaaaaaaaaaaa.md got", res)
	}
}

func TestUnicodeAwareTrimExactSizeAscii(t *testing.T) {
	tmp := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.md"
	res := unicodeAwareTrim(tmp, len(tmp))
	if res != tmp {
		t.Errorf("expected %s got %s", tmp, res)
	}
}

func TestUnicodeAwareTrimUnicode(t *testing.T) {
	tmp := "中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文.md"
	res := unicodeAwareTrim(tmp, shortFormatFileTruncate)
	if res != "~文中文中文中文中文中文中文.md" {
		t.Error("expected ~文中文中文中文中文中文中文.md got", res)
	}
}

func TestUnicodeAwareRightPad(t *testing.T) {
	tmp := unicodeAwareRightPad("", 10)
	if runewidth.StringWidth(tmp) != 10 {
		t.Errorf("expected length of 10")
	}
}

func TestUnicodeAwareRightPadUnicode(t *testing.T) {
	tmp := unicodeAwareRightPad("中文", 10)
	if runewidth.StringWidth(tmp) != 10 {
		t.Errorf("expected length of 10")
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
