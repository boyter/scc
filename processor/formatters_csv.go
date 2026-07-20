// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"cmp"
	"encoding/csv"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func toCSV(input chan *FileJob) string {
	if Files {
		return toCSVFiles(input)
	}

	return toCSVSummary(input)
}

func toCSVSummary(input chan *FileJob) string {
	language := aggregateLanguageSummary(input)
	language = sortLanguageSummary(language)

	record := []string{
		"Language",
		"Lines",
		"Code",
		"Comments",
		"Blanks",
		"Complexity",
		"Bytes",
		"Files",
		"ULOC",
	}
	if Cognitive {
		record = append(record, "Cognitive")
	}

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	_ = w.Write(record)

	for _, result := range language {
		record[0] = result.Name
		record[1] = strconv.FormatInt(result.Lines, 10)
		record[2] = strconv.FormatInt(result.Code, 10)
		record[3] = strconv.FormatInt(result.Comment, 10)
		record[4] = strconv.FormatInt(result.Blank, 10)
		record[5] = strconv.FormatInt(result.Complexity, 10)
		record[6] = strconv.FormatInt(result.Bytes, 10)
		record[7] = strconv.FormatInt(result.Count, 10)
		record[8] = strconv.Itoa(len(ulocLanguageCount[result.Name]))
		if Cognitive {
			record[9] = strconv.FormatInt(result.Cognitive, 10)
		}
		_ = w.Write(record)
	}

	w.Flush()

	return b.String()
}

func getCSVFilesSortFunc(sortBy string) func(a, b []string) int {
	// Cater for the common case of adding plural even for those options that don't make sense
	// as it's quite common for those who English is not a first language to make a simple mistake
	switch sortBy {
	case "name", "names":
		return func(a, b []string) int {
			return strings.Compare(a[2], b[2])
		}
	case "language", "languages", "lang", "langs":
		return func(a, b []string) int {
			return strings.Compare(a[0], b[0])
		}
	case "line", "lines":
		return func(a, b []string) int {
			i1, _ := strconv.ParseInt(a[3], 10, 64)
			i2, _ := strconv.ParseInt(b[3], 10, 64)
			return cmp.Compare(i2, i1)
		}
	case "blank", "blanks":
		return func(a, b []string) int {
			i1, _ := strconv.ParseInt(a[6], 10, 64)
			i2, _ := strconv.ParseInt(b[6], 10, 64)
			return cmp.Compare(i2, i1)
		}
	case "code", "codes":
		return func(a, b []string) int {
			i1, _ := strconv.ParseInt(a[4], 10, 64)
			i2, _ := strconv.ParseInt(b[4], 10, 64)
			return cmp.Compare(i2, i1)
		}
	case "comment", "comments":
		return func(a, b []string) int {
			i1, _ := strconv.ParseInt(a[5], 10, 64)
			i2, _ := strconv.ParseInt(b[5], 10, 64)
			return cmp.Compare(i2, i1)
		}
	case "complexity", "complexitys":
		// The cognitive value is appended as column 10 only when --cognitive is
		// set; sort by the active metric so the "complexity" key follows it.
		idx := 7
		if Cognitive {
			idx = 10
		}
		return func(a, b []string) int {
			i1, _ := strconv.ParseInt(a[idx], 10, 64)
			i2, _ := strconv.ParseInt(b[idx], 10, 64)
			return cmp.Compare(i2, i1)
		}
	case "cognitive", "cognitives":
		if !Cognitive {
			// No cognitive column emitted without --cognitive; fall back to name.
			return func(a, b []string) int {
				return strings.Compare(a[2], b[2])
			}
		}
		return func(a, b []string) int {
			i1, _ := strconv.ParseInt(a[10], 10, 64)
			i2, _ := strconv.ParseInt(b[10], 10, 64)
			return cmp.Compare(i2, i1)
		}
	case "byte", "bytes":
		return func(a, b []string) int {
			i1, _ := strconv.ParseInt(a[8], 10, 64)
			i2, _ := strconv.ParseInt(b[8], 10, 64)
			return cmp.Compare(i2, i1)
		}
	default:
		return func(a, b []string) int {
			return strings.Compare(a[2], b[2])
		}
	}
}

func toCSVFiles(input chan *FileJob) string {
	records := [][]string{}

	for result := range input {
		row := []string{
			result.Language,
			result.Location,
			result.Filename,
			strconv.FormatInt(result.Lines, 10),
			strconv.FormatInt(result.Code, 10),
			strconv.FormatInt(result.Comment, 10),
			strconv.FormatInt(result.Blank, 10),
			strconv.FormatInt(result.Complexity, 10),
			strconv.FormatInt(result.Bytes, 10),
			strconv.Itoa(result.Uloc),
		}
		if Cognitive {
			row = append(row, strconv.FormatInt(result.Cognitive, 10))
		}
		records = append(records, row)
	}

	slices.SortFunc(records, getCSVFilesSortFunc(SortBy))

	header := []string{
		"Language",
		"Provider",
		"Filename",
		"Lines",
		"Code",
		"Comments",
		"Blanks",
		"Complexity",
		"Bytes",
		"ULOC",
	}
	if Cognitive {
		header = append(header, "Cognitive")
	}
	recordsEnd := [][]string{header}

	recordsEnd = append(recordsEnd, records...)

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	_ = w.WriteAll(recordsEnd)
	w.Flush()

	return b.String()
}

// For very large repositories CSV stream can be used which prints results out as they come in
// with the express idea of lowering memory usage, see https://github.com/boyter/scc/issues/210 for
// the background on why this might be needed
func toCSVStream(input chan *FileJob) string {
	if Cognitive {
		fmt.Println("Language,Provider,Filename,Lines,Code,Comments,Blanks,Complexity,Bytes,Uloc,Cognitive")
	} else {
		fmt.Println("Language,Provider,Filename,Lines,Code,Comments,Blanks,Complexity,Bytes,Uloc")
	}

	var quoteRegex = regexp.MustCompile("\"")

	for result := range input {
		// Escape quotes in location and filename then surround with quotes.
		var location = "\"" + quoteRegex.ReplaceAllString(result.Location, "\"\"") + "\""
		var filename = "\"" + quoteRegex.ReplaceAllString(result.Filename, "\"\"") + "\""

		if Cognitive {
			fmt.Printf("%s,%s,%s,%d,%d,%d,%d,%d,%d,%d,%d\n",
				result.Language,
				location,
				filename,
				result.Lines,
				result.Code,
				result.Comment,
				result.Blank,
				result.Complexity,
				result.Bytes,
				result.Uloc,
				result.Cognitive,
			)
		} else {
			fmt.Printf("%s,%s,%s,%d,%d,%d,%d,%d,%d,%d\n",
				result.Language,
				location,
				filename,
				result.Lines,
				result.Code,
				result.Comment,
				result.Blank,
				result.Complexity,
				result.Bytes,
				result.Uloc,
			)
		}
	}

	return ""
}
