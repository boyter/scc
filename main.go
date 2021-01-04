// SPDX-License-Identifier: MIT
// SPDX-License-Identifier: Unlicense

package main

import (
	"fmt"
	"github.com/boyter/scc/processor"
	"github.com/spf13/cobra"
	"os"
)

//go:generate go run scripts/include.go
func main() {
	//f, _ := os.Create("scc.pprof")
	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()

	rootCmd := &cobra.Command{
		Use:     "scc",
		Short:   "scc [FILE or DIRECTORY]",
		Long:    fmt.Sprintf("Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation.\nVersion %s\nBen Boyter <ben@boyter.org> + Contributors", processor.Version),
		Version: processor.Version,
		Run: func(cmd *cobra.Command, args []string) {
			processor.DirFilePaths = args
			if processor.ConfigureLimits != nil {
				processor.ConfigureLimits()
			}
			processor.ConfigureGc()
			processor.ConfigureLazy(true)
			processor.Process()
		},
	}

	flags := rootCmd.PersistentFlags()

	flags.Int64Var(
		&processor.AverageWage,
		"avg-wage",
		56286,
		"average wage value used for basic COCOMO calculation",
	)
	flags.BoolVar(
		&processor.DisableCheckBinary,
		"binary",
		false,
		"disable binary file detection",
	)
	flags.BoolVar(
		&processor.Files,
		"by-file",
		false,
		"display output for every file",
	)
	flags.BoolVar(
		&processor.Ci,
		"ci",
		false,
		"enable CI output settings where stdout is ASCII",
	)
	flags.BoolVar(
		&processor.Ignore,
		"no-ignore",
		false,
		"disables .ignore file logic",
	)
	flags.BoolVar(
		&processor.GitIgnore,
		"no-gitignore",
		false,
		"disables .gitignore file logic",
	)
	flags.BoolVar(
		&processor.Debug,
		"debug",
		false,
		"enable debug output",
	)
	flags.StringSliceVar(
		&processor.PathDenyList,
		"exclude-dir",
		[]string{".git", ".hg", ".svn"},
		"directories to exclude",
	)
	flags.IntVar(
		&processor.GcFileCount,
		"file-gc-count",
		10000,
		"number of files to parse before turning the GC on",
	)
	flags.StringVarP(
		&processor.Format,
		"format",
		"f",
		"tabular",
		"set output format [tabular, wide, json, csv, csv-stream, cloc-yaml, html, html-table, sql, sql-insert]",
	)
	flags.StringSliceVarP(
		&processor.AllowListExtensions,
		"include-ext",
		"i",
		[]string{},
		"limit to file extensions [comma separated list: e.g. go,java,js]",
	)
	flags.BoolVarP(
		&processor.Languages,
		"languages",
		"l",
		false,
		"print supported languages and extensions",
	)
	flags.BoolVar(
		&processor.Cocomo,
		"no-cocomo",
		false,
		"remove COCOMO calculation output",
	)
	flags.BoolVar(
		&processor.Size,
		"no-size",
		false,
		"remove size calculation output",
	)
	flags.StringVar(
		&processor.SizeUnit,
		"size-unit",
		"si",
		"set size unit [si, binary, mixed, xkcd-kb, xkcd-kelly, xkcd-imaginary, xkcd-intel, xkcd-drive, xkcd-bakers]",
	)
	flags.BoolVarP(
		&processor.Complexity,
		"no-complexity",
		"c",
		false,
		"skip calculation of code complexity",
	)
	flags.BoolVarP(
		&processor.Duplicates,
		"no-duplicates",
		"d",
		false,
		"remove duplicate files from stats and output",
	)
	flags.BoolVarP(
		&processor.MinifiedGenerated,
		"min-gen",
		"z",
		false,
		"identify minified or generated files",
	)
	flags.BoolVarP(
		&processor.Minified,
		"min",
		"",
		false,
		"identify minified files",
	)
	flags.BoolVarP(
		&processor.Generated,
		"gen",
		"",
		false,
		"identify generated files",
	)
	flags.StringSliceVarP(
		&processor.GeneratedMarkers,
		"generated-markers",
		"",
		[]string{"do not edit"},
		"string markers in head of generated files",
	)
	flags.BoolVar(
		&processor.IgnoreMinifiedGenerate,
		"no-min-gen",
		false,
		"ignore minified or generated files in output (implies --min-gen)",
	)
	flags.BoolVar(
		&processor.IgnoreMinified,
		"no-min",
		false,
		"ignore minified files in output (implies --min)",
	)
	flags.BoolVar(
		&processor.IgnoreGenerated,
		"no-gen",
		false,
		"ignore generated files in output (implies --gen)",
	)
	flags.IntVar(
		&processor.MinifiedGeneratedLineByteLength,
		"min-gen-line-length",
		255,
		"number of bytes per average line for file to be considered minified or generated",
	)
	flags.StringArrayVarP(
		&processor.Exclude,
		"not-match",
		"M",
		[]string{},
		"ignore files and directories matching regular expression",
	)
	flags.StringVarP(
		&processor.FileOutput,
		"output",
		"o",
		"",
		"output filename (default stdout)",
	)
	flags.StringVarP(
		&processor.SortBy,
		"sort",
		"s",
		"files",
		"column to sort by [files, name, lines, blanks, code, comments, complexity]",
	)
	flags.BoolVarP(
		&processor.Trace,
		"trace",
		"t",
		false,
		"enable trace output (not recommended when processing multiple files)",
	)
	flags.BoolVarP(
		&processor.Verbose,
		"verbose",
		"v",
		false,
		"verbose output",
	)
	flags.BoolVarP(
		&processor.More,
		"wide",
		"w",
		false,
		"wider output with additional statistics (implies --complexity)",
	)
	flags.BoolVar(
		&processor.NoLarge,
		"no-large",
		false,
		"ignore files over certain byte and line size set by max-line-count and max-byte-count",
	)
	flags.BoolVar(
		&processor.IncludeSymLinks,
		"include-symlinks",
		false,
		"if set will count symlink files",
	)
	flags.Int64Var(
		&processor.LargeLineCount,
		"large-line-count",
		40000,
		"number of lines a file can contain before being removed from output",
	)
	flags.Int64Var(
		&processor.LargeByteCount,
		"large-byte-count",
		1000000,
		"number of bytes a file can contain before being removed from output",
	)
	flags.StringVar(
		&processor.CountAs,
		"count-as",
		"",
		"count extension as language [e.g. jsp:htm,chead:\"C Header\" maps extension jsp to html and chead to C Header]",
	)
	flags.StringVar(
		&processor.FormatMulti,
		"format-multi",
		"",
		"have multiple format output overriding --format [e.g. tabular:stdout,csv:file.csv,json:file.json]",
	)
	flags.StringVar(
		&processor.SQLProject,
		"sql-project",
		"",
		"use supplied name as the project identifier for the current run. Only valid with the --format sql or sql-insert option",
	)
	flags.StringVar(
		&processor.RemapUnknown,
		"remap-unknown",
		"",
		"inspect files of unknown type and remap by checking for a string and remapping the language [e.g. \"-*- C++ -*-\":\"C Header\"]",
	)
	flags.StringVar(
		&processor.RemapAll,
		"remap-all",
		"",
		"inspect every file and remap by checking for a string and remapping the language [e.g. \"-*- C++ -*-\":\"C Header\"]",
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
