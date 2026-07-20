// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mark3labs/mcp-go/mcp"

	jsoniter "github.com/json-iterator/go"
)

// makeCouplingRepo initialises a real on-disk git repo whose history couples
// alpha.go and beta.go: they change together in every commit, so the pair
// clears CouplingMinShared. gamma.go is touched once, so its pairs fall below
// the floor and never surface — giving both the all-pairs and per-file views a
// single, predictable coupling to assert on. Returns the repo path.
func makeCouplingRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	commits := []map[string]string{
		{"alpha.go": "package a\n// v0\n", "beta.go": "package b\n// v0\n"},
		{"alpha.go": "package a\n// v1\n", "beta.go": "package b\n// v1\n"},
		{"alpha.go": "package a\n// v2\n", "beta.go": "package b\n// v2\n", "gamma.go": "package g\n// v0\n"},
	}

	when := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i, snap := range commits {
		for path, content := range snap {
			full := filepath.Join(dir, path)
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				t.Fatalf("write %s: %v", full, err)
			}
			if _, err := wt.Add(path); err != nil {
				t.Fatalf("add %s: %v", path, err)
			}
		}
		_, err := wt.Commit("commit "+strconv.Itoa(i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Tester",
				Email: "tester@example.com",
				When:  when.Add(time.Duration(i) * time.Hour),
			},
		})
		if err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}
	return dir
}

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
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(nestedCognitiveSource), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return dir
}

// callCoupling invokes the coupling MCP handler with the given arguments and
// returns the result, failing the test on a transport-level (non-tool) error.
func callCoupling(t *testing.T, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = "coupling"
	req.Params.Arguments = args
	res, err := mcpCouplingHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned transport error: %v", err)
	}
	if res == nil {
		t.Fatal("handler returned nil result")
	}
	return res
}

// resultText concatenates the text content of a tool result.
func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

// TestMCPCouplingAllPairs: no `file` argument returns the repo-wide all-pairs
// report — distinguished by report:"coupling" and a top-level "pairs" array.
func TestMCPCouplingAllPairs(t *testing.T) {
	dir := makeCouplingRepo(t)

	res := callCoupling(t, map[string]any{"path": dir})
	if res.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, res))
	}

	var doc struct {
		Report string `json:"report"`
		Pairs  []struct {
			FileA  string `json:"fileA"`
			FileB  string `json:"fileB"`
			Shared int    `json:"shared"`
		} `json:"pairs"`
	}
	if err := jsoniter.Unmarshal([]byte(resultText(t, res)), &doc); err != nil {
		t.Fatalf("unmarshal all-pairs JSON: %v\n%s", err, resultText(t, res))
	}

	if doc.Report != "coupling" {
		t.Errorf("report = %q, want %q (per-file shape leaked into all-pairs mode)", doc.Report, "coupling")
	}
	if len(doc.Pairs) == 0 {
		t.Fatalf("expected at least one coupled pair, got none: %s", resultText(t, res))
	}
	// alpha.go and beta.go co-change in all three commits.
	p := doc.Pairs[0]
	if !((p.FileA == "alpha.go" && p.FileB == "beta.go") || (p.FileA == "beta.go" && p.FileB == "alpha.go")) {
		t.Errorf("top pair = (%s, %s), want the alpha.go/beta.go pair", p.FileA, p.FileB)
	}
	if p.Shared != 3 {
		t.Errorf("shared = %d, want 3", p.Shared)
	}
}

// TestMCPCouplingPerFile: with `file` set, the per-file blast-radius report is
// returned unchanged — report:"coupling-for" with a "target" and "partners".
func TestMCPCouplingPerFile(t *testing.T) {
	dir := makeCouplingRepo(t)

	res := callCoupling(t, map[string]any{"path": dir, "file": "alpha.go"})
	if res.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, res))
	}

	var doc struct {
		Report   string `json:"report"`
		Target   string `json:"target"`
		Partners []struct {
			File string `json:"file"`
		} `json:"partners"`
	}
	if err := jsoniter.Unmarshal([]byte(resultText(t, res)), &doc); err != nil {
		t.Fatalf("unmarshal per-file JSON: %v\n%s", err, resultText(t, res))
	}

	if doc.Report != "coupling-for" {
		t.Errorf("report = %q, want %q", doc.Report, "coupling-for")
	}
	if doc.Target != "alpha.go" {
		t.Errorf("target = %q, want %q", doc.Target, "alpha.go")
	}
	if len(doc.Partners) != 1 || doc.Partners[0].File != "beta.go" {
		t.Errorf("partners = %+v, want a single beta.go entry", doc.Partners)
	}
}

// TestMCPCouplingUnknownFile: an unknown `file` still surfaces the existing
// "not in HEAD" error rather than silently falling back to the all-pairs view.
func TestMCPCouplingUnknownFile(t *testing.T) {
	dir := makeCouplingRepo(t)

	res := callCoupling(t, map[string]any{"path": dir, "file": "does-not-exist.go"})
	if !res.IsError {
		t.Fatalf("expected error for unknown file, got success: %s", resultText(t, res))
	}
	msg := resultText(t, res)
	if !strings.Contains(msg, "not in HEAD") {
		t.Errorf("error = %q, want it to mention the target is not in HEAD", msg)
	}
	// The MCP caller passed a `file` argument and has never seen the CLI flag —
	// no flag name should leak into the message surfaced through MCP.
	if strings.Contains(msg, "--") {
		t.Errorf("error = %q, want no CLI flag names in the MCP-surfaced message", msg)
	}
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
