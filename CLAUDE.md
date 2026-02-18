# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

scc (Sloc, Cloc and Code) is a fast source code line counter written in Go. It counts lines of code, blank lines, comment lines across 322+ languages, and also calculates COCOMO cost estimates, cyclomatic complexity approximations, and ULOC (Unique Lines of Code) metrics. Module path: `github.com/boyter/scc/v3`.

## Common Commands

```bash
# Build
go build -ldflags="-s -w"

# Run all unit tests
go test ./...

# Run tests with race detector
go test -race -v ./...

# Run a single test
go test -run TestName ./processor/
go test -run TestName .          # for integration tests in main_test.go

# Code generation (required after editing languages.json)
go generate

# Format code
go fmt ./...

# Full test suite (generate, fmt, unit, race, integration, cross-compile)
./test-all.sh
```

## Architecture

### Processing Pipeline

The core is a three-stage concurrent pipeline connected by Go channels:

1. **File Discovery** (`processor.Process()`) — Uses `gocodewalker.NewParallelFileWalker()` to walk directories respecting `.gitignore`. Produces `potentialFilesQueue`.
2. **File Processing** (`fileProcessorWorker`) — N goroutines read files, detect language, and run `CountStats()`. Produces `fileSummaryJobQueue`.
3. **Summarization** (`fileSummarize`) — Aggregates results by language and formats output.

### Key Packages and Files

- **`main.go`** — CLI entry point using cobra. All flags bind to `processor.*` global variables.
- **`processor/`** — The entire core engine:
  - `processor.go` — `Process()` entry point, global config vars, language loading
  - `workers.go` — `CountStats()` state machine (the hot path), `fileProcessorWorker`
  - `structs.go` — All data structures: `FileJob`, `Language`, `LanguageFeature`, `Trie`
  - `constants.go` — **AUTO-GENERATED** from `languages.json` via `go generate`. Never edit manually.
  - `detector.go` — Language detection (extension mapping, shebang, keyword heuristics for ambiguous extensions)
  - `formatters.go` — Output formatters (tabular, wide, JSON, CSV, SQL, HTML, OpenMetrics, cloc-yaml)
  - `cocomo.go` — COCOMO cost estimation
- **`scripts/include.go`** — Code generator that reads `languages.json` and produces `processor/constants.go` using `scripts/languages.tmpl`
- **`languages.json`** — Source of truth for all language definitions (comment markers, string delimiters, complexity keywords, extensions, shebangs)

### Key Data Structures

- **`FileJob`** — Per-file state: content bytes, language, line/code/comment/blank counts, complexity, hash for duplicate detection
- **`Language`** — JSON-deserialized language rules (comment markers, string quotes, complexity keywords, extensions)
- **`LanguageFeature`** — Compiled/optimized runtime form of Language using Trie structures
- **`Trie`** — 256-way trie for O(n) multi-pattern matching of comment markers, string delimiters, and complexity keywords

### CountStats State Machine

The `CountStats()` function in `workers.go` is the performance-critical hot path. It processes each byte of a file through states: `SBlank`, `SCode`, `SComment`, `SCommentCode`, `SMulticomment`, `SMulticommentCode`, `SMulticommentBlank`, `SString`, `SDocString`.

## Adding/Modifying Languages

1. Edit `languages.json` in the repository root
2. Run `go generate` to regenerate `processor/constants.go`
3. Build with `go build`

## Integration Tests

`main_test.go` uses a re-invocation pattern: tests call the compiled test binary with flags to run `main()`, enabling end-to-end CLI testing without a separate build step. Test fixtures live in `examples/`.

## Public API

The processor package is a public API used by external projects. `ProcessConstants()` must be called once before using `CountStats()` on `FileJob` structs directly.
