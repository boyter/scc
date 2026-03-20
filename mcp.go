// SPDX-License-Identifier: MIT

package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/boyter/scc/v3/processor"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// mcpMu serializes MCP tool calls so concurrent requests
// don't race on processor package globals.
var mcpMu sync.Mutex

func startMCPServer() {
	mcpServer := server.NewMCPServer(
		"scc",
		processor.Version,
		server.WithToolCapabilities(false),
	)

	analyzeTool := mcp.NewTool("analyze",
		mcp.WithDescription(`Count lines of code, comments, blanks and estimate complexity for a project directory or file. Supports 200+ languages.

Returns per-language summary with:
- files: number of source files
- lines: total lines
- code: lines of actual code
- comment: lines of comments
- blank: blank lines
- complexity: estimated cyclomatic complexity
- bytes: total size in bytes

Also returns COCOMO cost/schedule estimates and optionally LOCOMO (LLM cost) estimates.

Use by_file with sort=complexity to find the most complex files in a project.`),
		mcp.WithString("path",
			mcp.Description("Directory or file path to analyze. Defaults to current directory."),
		),
		mcp.WithString("sort",
			mcp.Description("Column to sort results by: files, name, lines, blank, code, comment, complexity, bytes. Default: files."),
		),
		mcp.WithBoolean("by_file",
			mcp.Description("If true, return per-file results instead of per-language summary. Useful with sort to find e.g. the most complex or largest files. Use with limit to control response size."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of files to return per language when by_file is true. Defaults to 10. Set to -1 for unlimited."),
		),
		mcp.WithString("include_ext",
			mcp.Description("Comma-separated list of file extensions to include (e.g. 'go,java,js')."),
		),
		mcp.WithString("exclude_ext",
			mcp.Description("Comma-separated list of file extensions to exclude (e.g. 'json,xml')."),
		),
		mcp.WithBoolean("no_duplicates",
			mcp.Description("Remove duplicate files from stats."),
		),
		mcp.WithBoolean("no_min_gen",
			mcp.Description("Exclude minified or generated files."),
		),
		mcp.WithBoolean("locomo",
			mcp.Description("Include LOCOMO (LLM Output COst MOdel) cost estimation in results."),
		),
		mcp.WithString("locomo_preset",
			mcp.Description("LOCOMO model preset: large (GPT-4/Opus class), medium (Sonnet class), small (Haiku class), local (local LLM). Default: medium."),
		),
	)

	mcpServer.AddTool(analyzeTool, mcpAnalyzeHandler)

	errLogger := log.New(os.Stderr, "scc-mcp: ", log.LstdFlags)
	if err := server.ServeStdio(mcpServer, server.WithErrorLogger(errLogger)); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "scc-mcp: server error: %v\n", err)
		os.Exit(1)
	}
}

type mcpAnalyzeResponse struct {
	Path       string                     `json:"path"`
	Languages  []mcpLanguageResult        `json:"languages"`
	Totals     mcpTotals                  `json:"totals"`
	COCOMO     *mcpCOCOMO                 `json:"cocomo,omitempty"`
	LOCOMO     *mcpLOCOMO                 `json:"locomo,omitempty"`
	FileCount  int64                      `json:"totalFiles"`
	TotalLines int64                      `json:"totalLines"`
	TotalCode  int64                      `json:"totalCode"`
}

type mcpLanguageResult struct {
	Name       string          `json:"name"`
	Files      int64           `json:"files"`
	Lines      int64           `json:"lines"`
	Code       int64           `json:"code"`
	Comment    int64           `json:"comment"`
	Blank      int64           `json:"blank"`
	Complexity int64           `json:"complexity"`
	Bytes      int64           `json:"bytes"`
	FileList   []mcpFileResult `json:"fileList,omitempty"`
}

type mcpFileResult struct {
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

type mcpTotals struct {
	Files      int64 `json:"files"`
	Lines      int64 `json:"lines"`
	Code       int64 `json:"code"`
	Comment    int64 `json:"comment"`
	Blank      int64 `json:"blank"`
	Complexity int64 `json:"complexity"`
	Bytes      int64 `json:"bytes"`
}

type mcpCOCOMO struct {
	EstimatedCost           float64 `json:"estimatedCost"`
	EstimatedScheduleMonths float64 `json:"estimatedScheduleMonths"`
	EstimatedPeople         float64 `json:"estimatedPeople"`
}

type mcpLOCOMO struct {
	Cost                  float64 `json:"cost"`
	InputTokens           float64 `json:"inputTokens"`
	OutputTokens          float64 `json:"outputTokens"`
	GenerationSeconds     float64 `json:"generationSeconds"`
	ReviewHours           float64 `json:"reviewHours"`
	Preset                string  `json:"preset"`
	AverageComplexityMult float64 `json:"averageComplexityMultiplier"`
	Cycles                float64 `json:"cycles"`
}

func mcpAnalyzeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Extract parameters
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Verify path can be accessed
	if _, err := os.Stat(absPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path cannot be accessed: %s: %v", absPath, err)), nil
	}

	// Serialize access to processor globals so concurrent MCP
	// requests don't race on shared state.
	mcpMu.Lock()
	defer mcpMu.Unlock()

	// Configure processor globals for this request.
	// Some defaults are normally set by cobra flags which the MCP path
	// bypasses, so we set them explicitly here.
	processor.DirFilePaths = []string{absPath}
	processor.Format = "json"
	processor.Cocomo = false
	processor.Size = false
	processor.Files = false
	processor.PathDenyList = []string{".git", ".hg", ".svn"}
	processor.ExcludeFilename = []string{"package-lock.json", "Cargo.lock", "yarn.lock", "pubspec.lock", "Podfile.lock", "pnpm-lock.yaml"}

	if sortBy, ok := args["sort"].(string); ok && sortBy != "" {
		processor.SortBy = sortBy
	} else {
		processor.SortBy = "files"
	}

	if byFile, ok := args["by_file"].(bool); ok && byFile {
		processor.Files = true
	}

	fileLimit := 10 // default limit when by_file is true
	if l, ok := args["limit"].(float64); ok {
		if l < 0 {
			fileLimit = 0 // -1 (or any negative) means unlimited
		} else {
			fileLimit = int(l)
		}
	}

	if includeExt, ok := args["include_ext"].(string); ok && includeExt != "" {
		processor.AllowListExtensions = splitAndTrimExtensions(includeExt)
	} else {
		processor.AllowListExtensions = []string{}
	}

	if excludeExt, ok := args["exclude_ext"].(string); ok && excludeExt != "" {
		processor.ExcludeListExtensions = splitAndTrimExtensions(excludeExt)
	} else {
		processor.ExcludeListExtensions = []string{}
	}

	if noDups, ok := args["no_duplicates"].(bool); ok && noDups {
		processor.Duplicates = true
	} else {
		processor.Duplicates = false
	}

	if noMinGen, ok := args["no_min_gen"].(bool); ok && noMinGen {
		processor.IgnoreMinifiedGenerate = true
		// GeneratedMarkers is normally set by cobra flag defaults which
		// the MCP path bypasses, so set them here.
		if len(processor.GeneratedMarkers) == 0 {
			processor.GeneratedMarkers = []string{"do not edit", "<auto-generated />"}
		}
	} else {
		processor.IgnoreMinifiedGenerate = false
	}

	if locomo, ok := args["locomo"].(bool); ok && locomo {
		processor.Locomo = true
	} else {
		processor.Locomo = false
	}

	if locomoPreset, ok := args["locomo_preset"].(string); ok && locomoPreset != "" {
		processor.LocomoPresetName = locomoPreset
	} else {
		processor.LocomoPresetName = "medium"
	}

	processor.ConfigureLazy(true)

	// Run the analysis
	language, err := processor.ProcessResult()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	// Build response
	var totals mcpTotals
	langs := make([]mcpLanguageResult, 0, len(language))

	for _, l := range language {
		lr := mcpLanguageResult{
			Name:       l.Name,
			Files:      l.Count,
			Lines:      l.Lines,
			Code:       l.Code,
			Comment:    l.Comment,
			Blank:      l.Blank,
			Complexity: l.Complexity,
			Bytes:      l.Bytes,
		}

		if processor.Files && len(l.Files) > 0 {
			files := l.Files
			// Sort files within each language by the same criteria
			// used for languages so per-file output is ordered and
			// limit returns the top N rather than an arbitrary slice.
			sortFileJobs(files)
			if fileLimit > 0 && len(files) > fileLimit {
				files = files[:fileLimit]
			}
			lr.FileList = make([]mcpFileResult, 0, len(files))
			for _, f := range files {
				lr.FileList = append(lr.FileList, mcpFileResult{
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

		langs = append(langs, lr)

		totals.Files += l.Count
		totals.Lines += l.Lines
		totals.Code += l.Code
		totals.Comment += l.Comment
		totals.Blank += l.Blank
		totals.Complexity += l.Complexity
		totals.Bytes += l.Bytes
	}

	resp := mcpAnalyzeResponse{
		Path:       absPath,
		Languages:  langs,
		Totals:     totals,
		FileCount:  totals.Files,
		TotalLines: totals.Lines,
		TotalCode:  totals.Code,
	}

	// COCOMO estimate
	estimatedEffort := processor.EstimateEffort(totals.Code, processor.EAF)
	estimatedCost := processor.EstimateCost(estimatedEffort, processor.AverageWage, processor.Overhead)
	estimatedScheduleMonths := processor.EstimateScheduleMonths(estimatedEffort)
	estimatedPeople := 0.0
	if estimatedScheduleMonths > 0 {
		estimatedPeople = estimatedEffort / estimatedScheduleMonths
	}
	resp.COCOMO = &mcpCOCOMO{
		EstimatedCost:           estimatedCost,
		EstimatedScheduleMonths: estimatedScheduleMonths,
		EstimatedPeople:         estimatedPeople,
	}

	// LOCOMO estimate if requested
	if processor.Locomo {
		result := processor.LocomoEstimate(totals.Code, totals.Complexity)
		resp.LOCOMO = &mcpLOCOMO{
			Cost:                  result.Cost,
			InputTokens:           result.InputTokens,
			OutputTokens:          result.OutputTokens,
			GenerationSeconds:     result.GenerationSeconds,
			ReviewHours:           result.ReviewHours,
			Preset:                result.Preset,
			AverageComplexityMult: result.AverageComplexityMult,
			Cycles:                result.IterationFactor,
		}
	}

	// Serialize to JSON
	jsonBytes, err := jsonMarshal(resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// sortFileJobs sorts a slice of FileJob pointers using the current
// processor.SortBy value so that the most relevant files come first.
func sortFileJobs(files []*processor.FileJob) {
	switch processor.SortBy {
	case "name", "names", "language", "languages", "lang", "langs":
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return strings.Compare(a.Filename, b.Filename)
		})
	case "line", "lines":
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return cmp.Compare(b.Lines, a.Lines)
		})
	case "blank", "blanks":
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return cmp.Compare(b.Blank, a.Blank)
		})
	case "code", "codes":
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return cmp.Compare(b.Code, a.Code)
		})
	case "comment", "comments":
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return cmp.Compare(b.Comment, a.Comment)
		})
	case "complexity", "complexitys":
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return cmp.Compare(b.Complexity, a.Complexity)
		})
	case "byte", "bytes":
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return cmp.Compare(b.Bytes, a.Bytes)
		})
	default:
		slices.SortFunc(files, func(a, b *processor.FileJob) int {
			return cmp.Compare(b.Lines, a.Lines)
		})
	}
}

// splitAndTrimExtensions splits a comma-separated string into
// trimmed, non-empty extension entries.
func splitAndTrimExtensions(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
