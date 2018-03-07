package processor

import (
	"fmt"
	"github.com/ryanuber/columnize"
)

func fileSummerize(input *chan *FileJob) {
	output := []string{
		"-----",
		"Language | Files | Lines | Code | Comment | Blank | Byte",
		"-----",
	}

	languages := map[string]LanguageSummary{}

	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumByte int64 = 0, 0, 0, 0, 0, 0

	for res := range *input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumByte += res.Bytes

		_, ok := languages[res.Language]

		if !ok {
			languages[res.Language] = LanguageSummary{
				Name:    res.Language,
				Bytes:   res.Bytes,
				Lines:   res.Lines,
				Code:    res.Code,
				Comment: res.Comment,
				Blank:   res.Blank,
				Count:   1,
			}
		} else {
			tmp := languages[res.Language]

			languages[res.Language] = LanguageSummary{
				Name:    res.Language,
				Bytes:   tmp.Bytes + res.Bytes,
				Lines:   tmp.Lines + res.Lines,
				Code:    tmp.Code + res.Code,
				Comment: tmp.Comment + res.Comment,
				Blank:   tmp.Blank + res.Blank,
				Count:   tmp.Count + 1,
			}
		}
	}

	for name, summary := range languages {
		output = append(output, fmt.Sprintf("%s | %d | %d | %d | %d | %d | %d", name, summary.Count, summary.Lines, summary.Code, summary.Comment, summary.Blank, summary.Bytes))
	}

	output = append(output, "-----")
	output = append(output, fmt.Sprintf("Total | %d | %d | %d | %d | %d | %d", sumFiles, sumLines, sumCode, sumComment, sumBlank, sumByte))
	output = append(output, "-----")

	result := columnize.SimpleFormat(output)
	fmt.Println(result)
}
