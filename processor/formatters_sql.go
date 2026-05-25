// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func toSqlInsert(input chan *FileJob) string {
	str := &strings.Builder{}
	projectName := SQLProject
	if projectName == "" {
		projectName = strings.Join(DirFilePaths, ",")
	}

	var sumCode, sumComplexity int64
	str.WriteString("\nbegin transaction;")
	count := 0
	for res := range input {
		count++
		sumCode += res.Code
		sumComplexity += res.Complexity

		dir, _ := filepath.Split(res.Location)

		_, _ = fmt.Fprintf(str, "\ninsert into t values('%s', '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d, %d);",
			escapeSQLString(projectName),
			escapeSQLString(res.Language),
			escapeSQLString(res.Location),
			escapeSQLString(dir),
			escapeSQLString(res.Filename), res.Bytes, res.Blank, res.Comment, res.Code, res.Complexity, res.Uloc)

		// every 1000 files commit and start a new transaction to avoid overloading
		if count == 1000 {
			str.WriteString("\ncommit;")
			str.WriteString("\nbegin transaction;")
			count = 0
		}
	}
	str.WriteString("\ncommit;")

	cost, schedule, people := esstimateCostScheduleMonths(sumCode)
	currentTime := time.Now()
	es := float64(makeTimestampMilli()-startTimeMilli) * 0.001
	str.WriteString("\nbegin transaction;")
	_, _ = fmt.Fprintf(str, "\ninsert into metadata values('%s', '%s', %f, %f, %f, %f);",
		currentTime.Format("2006-01-02 15:04:05"),
		projectName,
		es,
		cost,
		schedule,
		people,
	)
	str.WriteString("\ncommit;")

	if Locomo {
		result := LocomoEstimate(sumCode, sumComplexity)
		str.WriteString("\nbegin transaction;")
		_, _ = fmt.Fprintf(str, "\ninsert into locomo_metadata values('%s', '%s', %f, %f, %f, %f, %f, '%s', %f);",
			currentTime.Format("2006-01-02 15:04:05"),
			projectName,
			result.Cost,
			result.InputTokens,
			result.OutputTokens,
			result.GenerationSeconds,
			result.ReviewHours,
			escapeSQLString(result.Preset),
			result.IterationFactor,
		)
		str.WriteString("\ncommit;")
	}

	return str.String()
}

// attempt to manually escape everything that could be a problem
func escapeSQLString(input string) string {
	var buffer bytes.Buffer
	for _, char := range input {
		switch char {
		case '\x00':
			// Remove null characters
			continue
		case '\'':
			// Escape single quote with another single quote
			buffer.WriteRune('\'')
			buffer.WriteRune('\'')
		default:
			buffer.WriteRune(char)
		}
	}
	return buffer.String()
}

func toSql(input chan *FileJob) string {
	var str strings.Builder

	str.WriteString(`create table metadata (   -- github.com/boyter/scc v ` + Version + `
             timestamp text,
             Project   text,
             elapsed_s real,
             estimated_cost real,
             estimated_schedule_months real,
             estimated_people real);
create table t        (
             Project       text   ,
             Language      text   ,
             File          text   ,
             File_dirname  text   ,
             File_basename text   ,
             nByte         integer,
             nBlank        integer,
             nComment      integer,
             nCode         integer,
             nComplexity   integer,
             nUloc         integer    
);`)

	str.WriteString(toSqlInsert(input))
	return str.String()
}
