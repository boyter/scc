# scc 4.0.0 Release Notes

This is a major release. It bundles new modes of operation (MCP, git
processing), new cost/output models, broader language support, and a handful of
behavioural changes to flag parsing, language detection and config handling that
can produce different results than 3.7.0 for the same input — hence the major
version bump.

## ⚠️ Breaking / behavioural changes

These can change scc's output or behaviour for an identical invocation against
3.7.0. Review before upgrading in scripts/CI.

- **Last duplicate flag now wins** (#723) — pflag parsing changed so that when a
  flag is supplied more than once on the command line, the last occurrence takes
  precedence. Scripts relying on the previous precedence may behave differently.
- **Linguist-inspired language detection** (#721) — files may now be classified
  as different languages than before, shifting per-language counts and the
  contents of JSON/CSV output.
- **Per-user and per-project config overrides** (#724) — scc now reads config
  overrides from user/project locations. A repository containing a config file
  will behave differently than under 3.7.0 for the same command.

## ✨ New features

- **MCP server support** (#686) — run scc as a Model Context Protocol server over
  stdio (`--mcp`), exposing an `analyze` tool for programmatic/LLM consumers.
  Includes sorting/limit tuning and shared settings with the CLI path.
- **Git processing** (#707) — integrated git handling, with `gc` enabled for git
  operations and help text highlighting the git options.
- **LOCOMO — LLM Output Cost Model** (#682) — a new cost calculation aimed at
  estimating LLM output cost, with configurable cycles.
- **Infographic report output** (#709) — new report/infographic rendering.
- **Percentage outputs in JSON** (#720) — JSON output now includes percentage
  figures.
- **Codeberg support** (#866376f) — Codeberg as a supported badge/target.
- **External ignore files via `--ignore-file`** — supply additional gitignore-format
  ignore files from outside the scanned tree (repeatable, order-sensitive, anchored
  at the scan root; any in-tree `.gitignore`/`.ignore`/`.sccignore` takes precedence).
  Resolves the long-standing request for global/shared ignore support without scc
  shelling out to git — point it at `~/.config/git/ignore` directly, or set it once
  in a `.scc` file via `SCC_CONFIG_PATH`. Backed by `CustomIgnoreFiles` in gocodewalker.

## 🗣 Language support

- Linguist-inspired language detection (#721).
- New / updated languages: Mojo (#685), Zen C (#708), XHTML (#681),
  IEC61131-3 Siemens extensions (#702), Mog (#683), Phoenix LiveView fix (#722).
- Patch/diff: recognize `.diff` files as Patch (#705).
- Rust: count the `?` try operator toward complexity (#701).
- C/C++ language information updated (#691); CUDA updated (#693).
- New "move" classifier (#703).
- Python: prevent apostrophes in docstrings from breaking parsing (#695); extra
  docstring test coverage.

## 🐛 Fixes

- Fix potential MCP issue (a148002) and MCP settings/sorting/limit tweaks.
- Fix circular symlinks bug (#694).
- Catch error when mapping to an unknown language (30fa28a); fix a panic
  (dc8c92e).
- Fix for issue #412 (#719).
- WIP/fix for issue #531 (#714).

## 🏎 Performance & internals

- Reduce string allocations (#687).
- Minor performance tweaks (#711) and general cleanup/optimization (#680).
- Apply `go fix` suggestions (#712); convert more shell tests to Go tests (#696).
- Automate file generation and add CI tests (#688, #706); update to latest Go
  and refresh dependencies.

## 🗒 To discuss / follow-up

- **Remaining `git` subprocess shell-out** — `remoteOriginName` in
  `processor/report.go` still runs `git config --get remote.origin.url` to derive
  the HTML report title. This is the only place scc shells out to the `git` binary
  (the rest use the go-git library or `gocodewalker.FindRepositoryRoot`). Decide
  whether to drop it (fall back to the path basename) or read it via go-git's config
  instead, so scc has no hard dependency on a `git` binary being installed.

## LineJudge Conformance and LineBench Representation (#769)

Reference notes on SCC's representation in the [LineJudge](https://github.com/loc-conformance/linejudge) conformance suite and the [LineBench](https://github.com/loc-conformance/linebench) benchmarking suite.

### Dialect Representation in LineJudge (`dialects/scc/default.toml`)

LineJudge defines counter rules through dialect files declaring how lines map into code, comments, or blanks:
- **Code**: Any line containing non-whitespace outside strings and comments, or plain string literals.
- **Comments**: Any line containing a single-line or multi-line comment with no non-whitespace outside the comment, or docstrings (such as Python triple-quoted docstrings) with no surrounding code on that line.
- **Blanks**: Any line containing only whitespace outside strings and comments.
- **Embedded Languages**: Rust documentation comments count as comments of the Rust file; Vue `<template>` sections count as part of the Vue file.

### Counting Heuristics and Tradeoffs

1. **Prefix-based comment tokens**:
   SCC's trie scanner matches comment tokens at line head. In languages with keyword comments (such as Batch `REM`), lines matching the token prefix (e.g., `REM comment` vs commands starting with `REM` like `REMOVE`) are classified based on prefix rules.
2. **Specialized Counters vs Generic Loop**:
   Language-specific fast path counters (Go, C, C++, C#, Java, Python, Rust, JavaScript, TypeScript, PHP, Ruby, etc.) optimize token matching and newline scanning while matching generic loop counting semantics.
3. **Deliberate Divergences**:
   Specific edge cases are pinned in `counterDivergences`:
   - JavaScript/TypeScript: Regular expression literals containing quotes or comment openers (`7010-regex_literal_holding_a_quote`, `7020-regex_holding_a_comment_opener`).
   - Rust: Character literals containing quotes or escape sequences (`4010-char_literal_holding_a_quote`, `4020-two_char_literals_holding_escapes`).

### Benchmark Representation in LineBench (`counters/scc.toml`)

- **Equal Work Invocations**: Standard benchmark runs use `--no-complexity` (`-c`) to establish baseline counting speed against counters that lack complexity analysis.
- **Ignore Rules**: By default, SCC utilizes `.gitignore`, `.ignore`, and `.sccignore` via `gocodewalker`. Multi-counter comparative tests should normalize ignore options or run against fixed file manifests.

---

_Generated from the commit history between `v3.7.0` and the 4.0.0 release._
