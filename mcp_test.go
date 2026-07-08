// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// nestedCognitiveSource is a small Go file whose branch keywords are nested,
// so cognitive complexity (nesting-weighted) exceeds plain cyclomatic
// complexity when the metric is enabled.
const nestedCognitiveSource = `package sample

func deeplyNested(items []int) int {
	total := 0
	for _, v := range items {
		if v > 0 {
			for i := 0; i < v; i++ {
				if i%2 == 0 {
					total += i
				}
			}
		}
	}
	return total
}
`

// callAnalyze runs the MCP analyze handler against dir with the supplied args
// and returns the decoded response.
func callAnalyze(t *testing.T, dir string, args map[string]any) mcpAnalyzeResponse {
	t.Helper()

	if args == nil {
		args = map[string]any{}
	}
	args["path"] = dir

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "analyze",
			Arguments: args,
		},
	}

	result, err := mcpAnalyzeHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("mcpAnalyzeHandler returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("mcpAnalyzeHandler returned tool error: %+v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatalf("mcpAnalyzeHandler returned no content")
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var resp mcpAnalyzeResponse
	if err := json.Unmarshal([]byte(text.Text), &resp); err != nil {
		t.Fatalf("failed to decode analyze response: %v\n%s", err, text.Text)
	}
	return resp
}

func writeNestedFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(nestedCognitiveSource), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return dir
}

// TestMcpAnalyzeCognitiveEnabled verifies that requesting cognitive=true
// yields a non-zero cognitive metric at the per-language, per-file and totals
// levels, and that the totals equal the sum of the per-language values.
func TestMcpAnalyzeCognitiveEnabled(t *testing.T) {
	dir := writeNestedFixture(t)

	resp := callAnalyze(t, dir, map[string]any{
		"cognitive": true,
		"by_file":   true,
	})

	if len(resp.Languages) == 0 {
		t.Fatalf("expected at least one language in response")
	}

	var sumLang int64
	for _, l := range resp.Languages {
		sumLang += l.Cognitive
	}

	if resp.Totals.Cognitive <= 0 {
		t.Fatalf("expected non-zero cognitive total, got %d", resp.Totals.Cognitive)
	}
	if sumLang != resp.Totals.Cognitive {
		t.Fatalf("totals cognitive (%d) != sum of per-language cognitive (%d)", resp.Totals.Cognitive, sumLang)
	}

	// Cognitive should exceed plain cyclomatic complexity for this nested input.
	if resp.Totals.Cognitive <= resp.Totals.Complexity {
		t.Fatalf("expected cognitive (%d) > complexity (%d) for nested source", resp.Totals.Cognitive, resp.Totals.Complexity)
	}

	// Per-file cognitive must be populated too.
	var fileCognitive int64
	for _, l := range resp.Languages {
		for _, f := range l.FileList {
			fileCognitive += f.Cognitive
		}
	}
	if fileCognitive != resp.Totals.Cognitive {
		t.Fatalf("sum of per-file cognitive (%d) != totals cognitive (%d)", fileCognitive, resp.Totals.Cognitive)
	}
}

// TestMcpAnalyzeCognitiveDisabled verifies that without the opt-in parameter
// the cognitive field is zero everywhere, leaving default output unchanged.
func TestMcpAnalyzeCognitiveDisabled(t *testing.T) {
	dir := writeNestedFixture(t)

	resp := callAnalyze(t, dir, map[string]any{
		"by_file": true,
	})

	if resp.Totals.Cognitive != 0 {
		t.Fatalf("expected zero cognitive total when not requested, got %d", resp.Totals.Cognitive)
	}
	for _, l := range resp.Languages {
		if l.Cognitive != 0 {
			t.Fatalf("expected zero cognitive for language %s, got %d", l.Name, l.Cognitive)
		}
		for _, f := range l.FileList {
			if f.Cognitive != 0 {
				t.Fatalf("expected zero cognitive for file %s, got %d", f.Filename, f.Cognitive)
			}
		}
	}
}

// TestMcpAnalyzeCognitiveNoLeak guards against per-call state accumulation:
// two consecutive analyze calls over the same input must return identical
// cognitive numbers (cf. the ProcessConstants MCP-leak class of bug).
func TestMcpAnalyzeCognitiveNoLeak(t *testing.T) {
	dir := writeNestedFixture(t)

	first := callAnalyze(t, dir, map[string]any{"cognitive": true})
	second := callAnalyze(t, dir, map[string]any{"cognitive": true})

	if first.Totals.Cognitive != second.Totals.Cognitive {
		t.Fatalf("cognitive total changed across calls: %d then %d", first.Totals.Cognitive, second.Totals.Cognitive)
	}
	if first.Totals.Complexity != second.Totals.Complexity {
		t.Fatalf("complexity total changed across calls: %d then %d", first.Totals.Complexity, second.Totals.Complexity)
	}
	if first.Totals.Code != second.Totals.Code {
		t.Fatalf("code total changed across calls: %d then %d", first.Totals.Code, second.Totals.Code)
	}
}
