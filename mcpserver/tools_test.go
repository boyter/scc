// SPDX-License-Identifier: MIT

package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func makeRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func TestAnalyzeProject(t *testing.T) {
	req := makeRequest("analyze_project", map[string]any{
		"path": "../examples/language",
	})

	result, err := handleAnalyzeProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var out analyzeResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if len(out.LanguageSummary) == 0 {
		t.Fatal("expected at least one language in summary")
	}
	if out.TotalFiles == 0 {
		t.Fatal("expected TotalFiles > 0")
	}
	if out.TotalCode == 0 {
		t.Fatal("expected TotalCode > 0")
	}
	if out.TotalLines == 0 {
		t.Fatal("expected TotalLines > 0")
	}
	if out.EstimatedCost == 0 {
		t.Fatal("expected EstimatedCost > 0")
	}
}

func TestAnalyzeProjectWithFiles(t *testing.T) {
	req := makeRequest("analyze_project", map[string]any{
		"path":          "../examples/language",
		"include_files": true,
	})

	result, err := handleAnalyzeProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var out analyzeResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	hasFiles := false
	for _, l := range out.LanguageSummary {
		if len(l.Files) > 0 {
			hasFiles = true
			break
		}
	}
	if !hasFiles {
		t.Fatal("expected per-file breakdown when include_files=true")
	}
}

func TestAnalyzeProjectSortBy(t *testing.T) {
	req := makeRequest("analyze_project", map[string]any{
		"path":    "../examples/language",
		"sort_by": "code",
	})

	result, err := handleAnalyzeProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var out analyzeResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// Verify sorted descending by code
	for i := 1; i < len(out.LanguageSummary); i++ {
		if out.LanguageSummary[i].Code > out.LanguageSummary[i-1].Code {
			t.Fatalf("expected descending sort by code, but index %d (%d) > index %d (%d)",
				i, out.LanguageSummary[i].Code, i-1, out.LanguageSummary[i-1].Code)
		}
	}
}

func TestAnalyzeProjectInvalidPath(t *testing.T) {
	req := makeRequest("analyze_project", map[string]any{
		"path": "/nonexistent/path/that/does/not/exist",
	})

	result, err := handleAnalyzeProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid path")
	}
}

func TestAnalyzeProjectMissingPathUsesDefault(t *testing.T) {
	old := projectDir
	projectDir = "../examples/language"
	defer func() { projectDir = old }()

	req := makeRequest("analyze_project", map[string]any{})

	result, err := handleAnalyzeProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var out analyzeResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if out.TotalFiles == 0 {
		t.Fatal("expected TotalFiles > 0 when using default projectDir")
	}
}

func TestAnalyzeProjectMissingPathNoDefault(t *testing.T) {
	old := projectDir
	projectDir = ""
	defer func() { projectDir = old }()

	req := makeRequest("analyze_project", map[string]any{})

	result, err := handleAnalyzeProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error when no path and no default directory")
	}
}

func TestCountFile(t *testing.T) {
	req := makeRequest("count_file", map[string]any{
		"path": "../examples/language/go.go",
	})

	result, err := handleCountFile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var out countFileResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if out.Language != "Go" {
		t.Fatalf("expected language Go, got %s", out.Language)
	}
	if out.Filename != "go.go" {
		t.Fatalf("expected filename go.go, got %s", out.Filename)
	}
	if out.Lines == 0 {
		t.Fatal("expected Lines > 0")
	}
	if out.Code == 0 {
		t.Fatal("expected Code > 0")
	}
	if out.Bytes == 0 {
		t.Fatal("expected Bytes > 0")
	}
}

func TestCountFileInvalidPath(t *testing.T) {
	req := makeRequest("count_file", map[string]any{
		"path": "/nonexistent/file.go",
	})

	result, err := handleCountFile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid path")
	}
}

func TestCountFileDirectory(t *testing.T) {
	req := makeRequest("count_file", map[string]any{
		"path": "../examples/language",
	})

	result, err := handleCountFile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error when path is a directory")
	}
}

func TestCountFileMissingPath(t *testing.T) {
	req := makeRequest("count_file", map[string]any{})

	result, err := handleCountFile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing path")
	}
}
