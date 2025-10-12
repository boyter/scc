// Tasks is a collection of tasks that would normally be done through shell scripts, but since it can be a real
// pain a lot of the time they have been put in here.

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func main() {
	var commentTags, languageCheck bool

	flag.BoolVar(&commentTags, "comment-tags", false, "check for comment codes to be fixed XXX, TODO, FIXME etc...")
	flag.BoolVar(&languageCheck, "language-check", false, "scc language check")
	flag.Parse()

	ran := false
	if commentTags {
		checkCommentTags()
		ran = true
	}
	if languageCheck {
		sccLanguageCheck()
		ran = true
	}

	// if nothing specified print the usage
	if !ran {
		flag.PrintDefaults()
	}
}

var pathDenyListPrefix = []string{"vendor/", ".git/"}

func checkCommentTags() {
	osExit := false
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		for _, prefix := range pathDenyListPrefix {
			if strings.HasPrefix(path, prefix) {
				return nil
			}
		}

		file, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tags := []string{"TODO", "FIXME", "HACK", "OPTIMIZE", "OPTIMISE", "XXX", "BUG", "DEPRECATED"}
		matches, err := findCommentTags(info.Name(), path, string(file), tags)
		if err != nil {
			return err
		}

		if len(matches) == 0 {
			return nil
		}

		osExit = true
		fmt.Printf("Found %d comment tag(s) in %s:\n", len(matches), matches[0].Path)
		for _, match := range matches {
			fmt.Printf("\t%s line %d. %s\n", match.Tag, match.Line, strings.TrimSpace(match.Comment))
		}

		return nil
	})

	if osExit {
		os.Exit(1)
	}
}

// CommentTagMatch holds information about a found comment tag.
type CommentTagMatch struct {
	Path    string // Original file path provided
	Line    int    // Line number where the tag was found
	Tag     string // The specific tag found (e.g., "TODO", "FIXME")
	Comment string // The full text of the comment line
}

// findCommentTags parses Go code and searches for specified tags within comments.
// It takes the filename (for position reporting), the original path (for context),
// the source code as a string, and a slice of tags to search for.
// Tags are matched case-sensitively at the beginning of a comment line,
// optionally followed by a colon or space. Assumes tagsToFind are in the desired case.
func findCommentTags(filename, path, code string, tagsToFind []string) ([]CommentTagMatch, error) {
	foundMatches := []CommentTagMatch{}
	fset := token.NewFileSet()

	// Parse the file, crucially including the ParseComments flag.
	// Using parser.AllErrors might report more syntax issues if the code isn't perfectly valid.
	// Using 0 (no mode flags beyond ParseComments) might be slightly faster if you only care about comments.
	f, err := parser.ParseFile(fset, filename, code, parser.ParseComments|parser.AllErrors)
	if err != nil {
		// It might be useful to know about parse errors even if we can partially process comments.
		// Depending on the error, f might still contain some AST data including comments.
		// Log the error but potentially continue if f is not nil.
		log.Printf("Warning: parsing error in %s: %v. Attempting to process comments anyway.", filename, err)
		// If f is nil, we cannot proceed.
		if f == nil {
			return nil, fmt.Errorf("fatal parsing error in %s: %w", filename, err)
		}
	}

	// Iterate through all comment groups found in the file.
	for _, commentGroup := range f.Comments {
		for _, comment := range commentGroup.List {
			commentText := comment.Text
			line := fset.Position(comment.Pos()).Line
			trimmedComment := ""

			// Handle both // and /* style comments
			if strings.HasPrefix(commentText, "//") {
				trimmedComment = strings.TrimSpace(commentText[2:])
			} else if strings.HasPrefix(commentText, "/*") {
				// For block comments, check the first line inside
				content := strings.TrimSpace(commentText[2:])
				// Remove trailing */ if it exists
				content = strings.TrimSuffix(content, "*/")
				content = strings.TrimSpace(content) // Trim space after removing */

				// Take only the first line of the block comment's content
				if idx := strings.Index(content, "\n"); idx != -1 {
					trimmedComment = strings.TrimSpace(content[:idx])
				} else {
					trimmedComment = content // It's a single-line block comment like /* TODO */
				}
			} else {
				continue // Should not happen with standard Go comments
			}

			if trimmedComment == "" {
				continue // Skip empty or whitespace-only comment lines
			}

			// Check if the comment starts with any of the target tags (case-sensitive)
			for _, tag := range tagsToFind {
				match := false
				// Check for "TAG:" or "TAG " prefix, or exact match "TAG"
				if strings.HasPrefix(trimmedComment, tag+":") {
					match = true
				} else if strings.HasPrefix(trimmedComment, tag+" ") {
					match = true
				} else if trimmedComment == tag { // Handle tag alone e.g. // TODO
					match = true
				}

				if match {
					foundMatches = append(foundMatches, CommentTagMatch{
						Path:    path,
						Line:    line,
						Tag:     tag,         // Use the tag directly as provided
						Comment: commentText, // Store the full original comment line
					})
					// Found a tag on this line, no need to check for others on the same line.
					// Remove 'break' if you want to find multiple tags on the same comment line
					// (e.g., "// TODO FIXME Add tests").
					break
				}
			}
		}
	}

	// Note: Comments attached directly to declarations (f.Decls) are not processed here.
	// This loop covers general comments. Add AST inspection if needed for doc comments.

	return foundMatches, nil // Return matches found, even if there were parsing errors logged earlier
}

func sccLanguageCheck() {
	specificLanguages := []string{
		"ABNF", "Alchemist", "Alloy", "Arturo", "Astro", "AWK", "BASH", "Bean", "Bicep",
		"Bitbucket Pipeline", "Blueprint", "Boo", "Bosque", "C3", "C Shell", "C#", "Cairo",
		"Cangjie", "Chapel", "Circom", "Clipper", "Clojure", "CMake", "Cuda", "DAML", "DM",
		"Docker ignore", "Dockerfile", "DOT", "Elixir Template", "Elm", "EmiT", "F#", "Factor",
		"Flow9", "FSL", "Futhark", "FXML", "Gemfile", "Gleam", "Go", "Go+", "Godot Scene",
		"GraphQL", "Gwion", "HAML", "Hare", "HCL", "ignore", "INI", "Java", "JavaScript",
		"JCL", "JSON5", "JSONC", "jq", "Korn Shell", "Koto", "LALRPOP", "License", "LiveScript",
		"LLVM IR", "Lua", "Luau", "Luna", "Makefile", "Metal", "Monkey C", "Moonbit", "Nushell",
		"OpenQASM", "OpenTofu", "Perl", "Pkl", "PostScript", "Proto", "Python", "Q#", "R",
		"Racket", "Rakefile", "RAML", "Redscript", "Scallop", "Shell", "Sieve", "Slang",
		"Slint", "Smalltalk", "Snakemake", "Stan", "Systemd", "Tact", "Teal", "Tera",
		"Templ", "Terraform", "TTCN-3", "TypeScript", "TypeSpec", "Typst", "Up", "Vala",
		"Web Services", "wenyan", "Wren", "XMake", "XML Schema", "YAML", "Yarn", "Zig",
		"ZoKrates", "Zsh",
	}

	// above should be in order but sort just in case
	slices.Sort(specificLanguages)

	executable := "./scc"
	cmdArgs := []string{"examples/language/", "--no-scc-ignore"}
	timeout := 10 * time.Second

	response, err := Execute(executable, cmdArgs, timeout)
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		os.Exit(1)
	}

	allPassed := true
	for _, lang := range specificLanguages {
		if strings.Contains(response.Stdout+" ", lang) {
			fmt.Printf("\033[32mPASSED %s Language Check\033[0m\n", lang)
		} else {
			fmt.Printf("\033[31m=======================================================\n")
			fmt.Printf("FAILED Should be able to find %s\n", lang)
			fmt.Printf("=======================================================\033[0m\n")
			allPassed = false
		}
	}

	if !allPassed {
		os.Exit(1)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ExecuteResponse struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

var TimeoutError = errors.New("process killed due to timeout")

// Execute attempts to run a command via shell execution with supplied arguments and a time duration
// the duration if exceeded will have the underlying task attempt to be killed
// in all cases the current state of the stdout and stderr will be returned along with the time taken
// Note that it is possible that the duration returned exceeds the timeout in the case of misbehaving
// processes which do not respond to the kill signal correctly
func Execute(executable string, cmdArgs []string, timeout time.Duration) (ExecuteResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.Command(executable, cmdArgs...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	start := time.Now()
	exitCode := 0
	if err := cmd.Start(); err != nil {
		exitCode = -1 // set to -1 if we can't determine the exit code
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return ExecuteResponse{
			ExitCode: exitCode,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			Duration: time.Since(start),
		}, err
	}

	// Create a channel to monitor command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	select {
	case <-ctx.Done():
		exitCode = -2 // set to -2 to indicate process was killed
		err = errors.Join(TimeoutError, cmd.Process.Kill())
	case err = <-done:
		// ensure we capture the exit code
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1 // Fallback if exit code is unavailable
			}
		} else {
			exitCode = 0
		}
	}

	return ExecuteResponse{
		ExitCode: exitCode,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: time.Since(start),
	}, err
}
