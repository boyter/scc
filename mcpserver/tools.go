// SPDX-License-Identifier: MIT

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/boyter/scc/v3/processor"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// processorMu serializes analyze_project calls because the processor package
// uses global variables for configuration and state.
var processorMu sync.Mutex

func registerTools(s *server.MCPServer) {
	analyzeProjectTool := mcp.NewTool("analyze_project",
		mcp.WithDescription("Run scc on a directory or file and return aggregated language stats plus COCOMO cost estimates"),
		mcp.WithString("path",
			mcp.Description("Directory or file path to analyze (defaults to the project directory)"),
		),
		mcp.WithBoolean("include_files",
			mcp.Description("Include per-file breakdown in results"),
		),
		mcp.WithString("sort_by",
			mcp.Description("Sort order: files, name, lines, code, comments, blanks, complexity"),
		),
	)

	countFileTool := mcp.NewTool("count_file",
		mcp.WithDescription("Analyze a single file and return line counts, complexity, and language detection"),
		mcp.WithString("path",
			mcp.Description("File path to analyze"),
			mcp.Required(),
		),
	)

	s.AddTool(analyzeProjectTool, handleAnalyzeProject)
	s.AddTool(countFileTool, handleCountFile)
}

type analyzeResult struct {
	LanguageSummary         []langSummary `json:"languageSummary"`
	TotalFiles              int64         `json:"totalFiles"`
	TotalLines              int64         `json:"totalLines"`
	TotalCode               int64         `json:"totalCode"`
	TotalComment            int64         `json:"totalComment"`
	TotalBlank              int64         `json:"totalBlank"`
	TotalBytes              int64         `json:"totalBytes"`
	TotalComplexity         int64         `json:"totalComplexity"`
	EstimatedCost           float64       `json:"estimatedCost"`
	EstimatedScheduleMonths float64       `json:"estimatedScheduleMonths"`
	EstimatedPeople         float64       `json:"estimatedPeople"`
}

type langSummary struct {
	Name       string     `json:"name"`
	Bytes      int64      `json:"bytes"`
	Lines      int64      `json:"lines"`
	Code       int64      `json:"code"`
	Comment    int64      `json:"comment"`
	Blank      int64      `json:"blank"`
	Complexity int64      `json:"complexity"`
	Count      int64      `json:"count"`
	Files      []fileInfo `json:"files,omitempty"`
}

type fileInfo struct {
	Location   string `json:"location"`
	Filename   string `json:"filename"`
	Language   string `json:"language"`
	Lines      int64  `json:"lines"`
	Code       int64  `json:"code"`
	Comment    int64  `json:"comment"`
	Blank      int64  `json:"blank"`
	Complexity int64  `json:"complexity"`
	Bytes      int64  `json:"bytes"`
}

func handleAnalyzeProject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("path", "")
	if path == "" {
		path = projectDir
	}
	if path == "" {
		return errResult("path is required (no default project directory configured)"), nil
	}

	includeFiles := req.GetBool("include_files", false)
	sortBy := req.GetString("sort_by", "files")

	processorMu.Lock()
	defer processorMu.Unlock()

	resetProcessorGlobals()

	processor.DirFilePaths = []string{path}
	processor.SortBy = sortBy
	processor.Files = includeFiles
	processor.Verbose = false
	processor.Debug = false
	processor.Trace = false

	result, err := processor.ProcessToResult()
	if err != nil {
		return errResult(err.Error()), nil
	}

	langs := make([]langSummary, 0, len(result.LanguageSummary))
	for _, l := range result.LanguageSummary {
		ls := langSummary{
			Name:       l.Name,
			Bytes:      l.Bytes,
			Lines:      l.Lines,
			Code:       l.Code,
			Comment:    l.Comment,
			Blank:      l.Blank,
			Complexity: l.Complexity,
			Count:      l.Count,
		}
		if includeFiles && len(l.Files) > 0 {
			ls.Files = make([]fileInfo, 0, len(l.Files))
			for _, f := range l.Files {
				ls.Files = append(ls.Files, fileInfo{
					Location:   f.Location,
					Filename:   f.Filename,
					Language:   f.Language,
					Lines:      f.Lines,
					Code:       f.Code,
					Comment:    f.Comment,
					Blank:      f.Blank,
					Complexity: f.Complexity,
					Bytes:      f.Bytes,
				})
			}
		}
		langs = append(langs, ls)
	}

	out := analyzeResult{
		LanguageSummary:         langs,
		TotalFiles:              result.TotalFiles,
		TotalLines:              result.TotalLines,
		TotalCode:               result.TotalCode,
		TotalComment:            result.TotalComment,
		TotalBlank:              result.TotalBlank,
		TotalBytes:              result.TotalBytes,
		TotalComplexity:         result.TotalComplexity,
		EstimatedCost:           result.EstimatedCost,
		EstimatedScheduleMonths: result.EstimatedScheduleMonths,
		EstimatedPeople:         result.EstimatedPeople,
	}

	b, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(b)), nil
}

type countFileResult struct {
	Filename   string `json:"filename"`
	Language   string `json:"language"`
	Lines      int64  `json:"lines"`
	Code       int64  `json:"code"`
	Comment    int64  `json:"comment"`
	Blank      int64  `json:"blank"`
	Complexity int64  `json:"complexity"`
	Bytes      int64  `json:"bytes"`
}

func handleCountFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return errResult("path is required"), nil
	}

	if !filepath.IsAbs(path) && projectDir != "" {
		path = filepath.Join(projectDir, path)
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return errResult(fmt.Sprintf("cannot read file: %s", path)), nil
	}
	if info.IsDir() {
		return errResult("path is a directory, not a file"), nil
	}

	// Ensure language features are loaded (idempotent after first call)
	processor.ProcessConstants()

	content, err := os.ReadFile(path)
	if err != nil {
		return errResult(fmt.Sprintf("error reading file: %v", err)), nil
	}

	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	if ext != "" {
		ext = ext[1:] // remove leading dot
	}

	possibleLanguages := processor.ExtensionToLanguage[ext]

	job := &processor.FileJob{
		Filename:          filename,
		Extension:         ext,
		Location:          path,
		Content:           content,
		Bytes:             info.Size(),
		PossibleLanguages: possibleLanguages,
	}

	job.Language = processor.DetermineLanguage(filename, job.Language, job.PossibleLanguages, content)
	processor.CountStats(job)

	if job.Binary {
		return errResult("file identified as binary"), nil
	}

	out := countFileResult{
		Filename:   filename,
		Language:   job.Language,
		Lines:      job.Lines,
		Code:       job.Code,
		Comment:    job.Comment,
		Blank:      job.Blank,
		Complexity: job.Complexity,
		Bytes:      job.Bytes,
	}

	b, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(b)), nil
}

func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: msg,
			},
		},
		IsError: true,
	}
}

// resetProcessorGlobals resets the processor package's global state so
// successive analyze_project calls don't accumulate stale data.
func resetProcessorGlobals() {
	processor.DirFilePaths = []string{}
	processor.Files = false
	processor.Verbose = false
	processor.Debug = false
	processor.Trace = false
	processor.Duplicates = false
	processor.Complexity = false
	processor.Cocomo = false
	processor.Size = false
	processor.SortBy = ""
	processor.Format = ""
	processor.FileOutput = ""
	processor.Exclude = []string{}
	processor.AllowListExtensions = []string{}
	processor.ExcludeListExtensions = []string{}
	processor.ExcludeFilename = []string{}
	processor.CountAs = ""
	processor.RemapUnknown = ""
	processor.RemapAll = ""
	processor.UlocMode = false
	processor.Dryness = false
	processor.MinifiedGenerated = false
	processor.Minified = false
	processor.Generated = false
	processor.IgnoreMinifiedGenerate = false
	processor.IgnoreMinified = false
	processor.IgnoreGenerated = false
	processor.NoLarge = false
	processor.More = false
	processor.CocomoProjectType = "organic"
	processor.AverageWage = 56286
	processor.Overhead = 2.4
	processor.EAF = 1.0
	processor.CurrencySymbol = "$"
	processor.PathDenyList = []string{".git", ".hg", ".svn"}
	processor.GitIgnore = false
	processor.Ignore = false
	processor.GitModuleIgnore = false
	processor.SccIgnore = false
	processor.DisableCheckBinary = false
	processor.IncludeSymLinks = false
	processor.FormatMulti = ""
	processor.Percent = false
	processor.MaxMean = false
}
