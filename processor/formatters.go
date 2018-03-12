package processor

import (
	"fmt"
	"github.com/ryanuber/columnize"
	// "sort"
)

// TODO write our own formatter code becuase columnize is actually too slow for our purposes
// since it requires that we loop over the results again in order to work out the sizes which
// we already know because they should be no longer than the summary
// For the files summary it slows things down from ~5 seconds to ~10 seconds
// Also needs to support sorting of values actually...
// Maybe have a guesser which guesses the size and if it gets it right output looks good
// otherwise it just takes longer to get it right... guesses could be pretty accurate

func fileSummerizeFiles(input *chan *FileJob) {
	output := []string{
		"-----",
		"Language | Files | Lines | Code | Comment | Blank | Complexity | Byte",
		"-----",
	}

	languages := map[string]LanguageSummary{}

	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumByte, sumComplexity int64 = 0, 0, 0, 0, 0, 0, 0

	for res := range *input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumByte += res.Bytes
		sumComplexity += res.Complexity

		_, ok := languages[res.Language]

		if !ok {

			files := []*FileJob{}
			files = append(files, res)

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Bytes:      res.Bytes,
				Lines:      res.Lines,
				Code:       res.Code,
				Comment:    res.Comment,
				Blank:      res.Blank,
				Complexity: res.Complexity,
				Count:      1,
				Files:      files,
			}

		} else {
			tmp := languages[res.Language]

			files := append(tmp.Files, res)

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Bytes:      tmp.Bytes + res.Bytes,
				Lines:      tmp.Lines + res.Lines,
				Code:       tmp.Code + res.Code,
				Comment:    tmp.Comment + res.Comment,
				Blank:      tmp.Blank + res.Blank,
				Complexity: tmp.Complexity + res.Complexity,
				Count:      tmp.Count + 1,
				Files:      files,
			}
		}
	}

	for name, summary := range languages {
		output = append(output, "-----")
		output = append(output, fmt.Sprintf("%s | %d | %d | %d | %d | %d | %d | %d", name, summary.Count, summary.Lines, summary.Code, summary.Comment, summary.Blank, summary.Complexity, summary.Bytes))
		output = append(output, "-----")
		for _, res := range summary.Files {

			tmp := res.Location

			if len(tmp) >= 31 {
				totrim := len(tmp) - 30
				tmp = "~" + tmp[totrim:]
			}

			output = append(output, fmt.Sprintf("%s |  | %d | %d | %d | %d | %d | %d", tmp, res.Lines, res.Code, res.Comment, res.Blank, res.Complexity, res.Bytes))
		}
	}

	output = append(output, "-----")
	output = append(output, fmt.Sprintf("Total | %d | %d | %d | %d | %d | %d | %d", sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumByte))
	output = append(output, "-----")

	result := columnize.SimpleFormat(output)
	fmt.Println(result)
}

func fileSummerize(input *chan *FileJob) {
	output := []string{
		"-----",
		"Language | Files | Lines | Code | Comment | Blank | Complexity | Byte",
		"-----",
	}

	languages := map[string]LanguageSummary{}

	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumByte, sumComplexity int64 = 0, 0, 0, 0, 0, 0, 0

	for res := range *input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumByte += res.Bytes
		sumComplexity += res.Complexity

		_, ok := languages[res.Language]

		if !ok {
			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Bytes:      res.Bytes,
				Lines:      res.Lines,
				Code:       res.Code,
				Comment:    res.Comment,
				Blank:      res.Blank,
				Complexity: res.Complexity,
				Count:      1,
			}
		} else {
			tmp := languages[res.Language]

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Bytes:      tmp.Bytes + res.Bytes,
				Lines:      tmp.Lines + res.Lines,
				Code:       tmp.Code + res.Code,
				Comment:    tmp.Comment + res.Comment,
				Blank:      tmp.Blank + res.Blank,
				Complexity: tmp.Complexity + res.Complexity,
				Count:      tmp.Count + 1,
			}
		}
	}

	// sort.Slice(languages, func(i, j int) bool {
	// 	return i.
	// })
	// Convert map to list then sort and done

	for _, summary := range languages {
		output = append(output, fmt.Sprintf("%s | %d | %d | %d | %d | %d | %d | %d", summary.Name, summary.Count, summary.Lines, summary.Code, summary.Comment, summary.Blank, summary.Complexity, summary.Bytes))
	}

	output = append(output, "-----")
	output = append(output, fmt.Sprintf("Total | %d | %d | %d | %d | %d | %d | %d", sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumByte))
	output = append(output, "-----")

	result := columnize.SimpleFormat(output)
	fmt.Println(result)
}
