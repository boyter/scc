// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/boyter/scc/v3/processor"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func printShellCompletion(cmd *cobra.Command, command string) error {
	switch command {
	case "bash":
		return cmd.GenBashCompletionV2(os.Stdout, true)
	case "zsh":
		return cmd.GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.GenPowerShellCompletion(os.Stdout)
	default:
		return errors.New("Unknown shell: " + command)
	}
}

func printFlagSuggestion(flagSet *pflag.FlagSet, unknownFlag string) {
	flags := processor.GetMostSimilarFlags(flagSet, unknownFlag)
	if len(flags) == 0 {
		return
	}

	if len(flags) > 1 {
		_, _ = fmt.Fprintf(os.Stderr, "The most similar flags of --%s are:\n", unknownFlag)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "The most similar flag of --%s is:\n", unknownFlag)
	}

	for _, flag := range flags {
		_, _ = fmt.Fprintf(os.Stderr, "\t--%s\n", flag)
	}
}

//go:generate go run scripts/include.go
func main() {
	// f, _ := os.Create("scc.pprof")
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	// Handle --mcp flag before cobra to avoid interfering with stdio. The MCP
	// server deliberately does not load global/project config (§3.3 step 0).
	if slices.Contains(os.Args[1:], "--mcp") {
		startMCPServer()
		return
	}

	// handle "scc @flags.txt" syntax. The sole-argument trigger is preserved;
	// only the splitter is swapped for the shared tokenizer, which adds comment
	// stripping, quote-aware tokenization and drops the old blank-line empty-arg
	// bug. @file keeps positional lines (allowPositional=true), see §6.
	if len(os.Args) == 2 && strings.HasPrefix(os.Args[1], "@") {
		filename := strings.TrimPrefix(os.Args[1], "@")
		b, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading flags from a file: %s\n", err)
			os.Exit(1)
		}
		os.Args = append([]string{os.Args[0]}, parseConfigArgs(string(b), true)...)
	}

	// The genuine CLI is the args the user actually typed, captured after @file
	// expansion (@file is trusted, §6) but before config tokens are prepended.
	// It is the sole source of the write vars (§5.2).
	genuineCLI := slices.Clone(os.Args[1:])

	// Pre-scan os.Args for the config-control flags, then discover and read the
	// global/project config sources (§3, §4).
	noConfig, findRoot, explicitPath := preScanConfig(os.Args)
	globalTokens, projectTokens, discoverErr := discoverConfigArgs(noConfig, findRoot, explicitPath)
	if discoverErr != nil {
		// Unreadable explicit source (--config / SCC_CONFIG_PATH): fatal (§8.1).
		processor.PrintError(discoverErr.Error())
		os.Exit(1)
	}

	// Fast path when no config was discovered: the merged list *is* the genuine
	// CLI, so bind write flags directly to the real vars and parse once, exactly
	// as scc does today. The discard / CLI-only split engages only when config is
	// actually present (§5.2 fast path).
	configPresent := len(globalTokens) != 0 || len(projectTokens) != 0

	// Build the merged argument list: [argv0] + global + project + CLI. Scalar
	// precedence (global < project < CLI) falls out of prepend + last-wins, and
	// slice flags union naturally (§3.3, §7).
	merged := make([]string, 0, 1+len(globalTokens)+len(projectTokens)+len(genuineCLI))
	merged = append(merged, os.Args[0])
	merged = append(merged, globalTokens...)
	merged = append(merged, projectTokens...)
	merged = append(merged, genuineCLI...)

	// Write-flag bindings for the merged parse. Config present -> discards (config
	// can never reach the real vars); no config -> the real vars directly.
	var discardOutput, discardReport, discardFormatMulti string
	bindings := &flagBindings{
		output:      &processor.FileOutput,
		report:      &processor.ReportOut,
		formatMulti: &processor.FormatMulti,
	}
	if configPresent {
		bindings.output = &discardOutput
		bindings.report = &discardReport
		bindings.formatMulti = &discardFormatMulti
	}

	rootCmd := &cobra.Command{
		Use:   "scc [flags] [files or directories]",
		Short: "scc [files or directories]",
		Long:  fmt.Sprintf("Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation.\nVersion %s\nBen Boyter <ben@boyter.org> + Contributors", processor.Version),
		Example: `  Count the current directory:
    scc

  Count a specific folder or file:
    scc myproject/
    scc main.go

  Count several paths at once:
    scc src/ docs/ README.md

  Show a per-file breakdown instead of the per-language summary:
    scc --by-file

  Output as CSV or JSON (e.g. for further processing):
    scc --format csv
    scc --format json -o counts.json

  Count an unrecognised extension as a known language:
    scc --count-as jsp:html

  Count files matching a path pattern as a new category (glob by default):
    scc --count-as-pattern '*_spec.rb:Ruby Spec:Ruby'

  Generate a self-contained HTML infographic report:
    scc --report
    scc --report=out.html --report-title "myrepo" --report-skip cocomo`,
		Version: processor.Version,
		Run: func(cmd *cobra.Command, args []string) {
			processor.DirFilePaths = args
			processor.ConfigureGc()
			processor.ConfigureLazy(true)

			// Detect if LOCOMO price/tps flags were explicitly set
			processor.LocomoInputPriceSet = cmd.PersistentFlags().Changed("locomo-input-price")
			processor.LocomoOutputPriceSet = cmd.PersistentFlags().Changed("locomo-output-price")
			processor.LocomoTPSSet = cmd.PersistentFlags().Changed("locomo-tps")
			processor.LocomoCyclesSet = cmd.PersistentFlags().Changed("locomo-cycles")

			if v, err := cmd.PersistentFlags().GetBool("no-fold-authors"); err == nil && v {
				processor.FoldAuthors = false
			}

			// Source the write vars from the genuine CLI alone (file output is a
			// CLI-only capability), then warn if config tried to set one (§5.2).
			if configPresent {
				resolveWriteFlags(genuineCLI)
				warnIfConfigWrote(cmd.PersistentFlags())
			}

			// Merge the built-in defaults back into the empty-defaulted slice
			// flags (§7), then flush any buffered config trace/debug (§8.0a).
			applySliceDefaults()
			flushConfigTrace()

			processor.Process()
		},
	}

	flags := rootCmd.PersistentFlags()
	registerFlags(flags, bindings)
	registerConfigControlFlags(flags)

	// If invoked in the format of "scc completion --shell [name of shell]", generate command line completions instead.
	// With the --shell option, unintentionally triggering shell completions should be highly unlikely. This reads the
	// genuine os.Args (config tokens are handed to cobra via SetArgs below, never prepended into os.Args), so a ./.scc
	// in the working directory cannot shift args[1] and break completion (§3.3 step 5).
	args := os.Args
	if len(args) == 4 && args[1] == "completion" && args[2] == "--shell" {
		err := printShellCompletion(rootCmd, args[3])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error printing shell completion: %s\n", err)
		}
		return
	}

	// Hand the merged list to cobra without mutating os.Args.
	rootCmd.SetArgs(merged[1:])

	if err := rootCmd.Execute(); err != nil {
		// If a flag does not exist and is not a shorthand, it may be a spelling error. Search for and print possible options.
		if notExistError, ok := err.(*pflag.NotExistError); ok && len(notExistError.GetSpecifiedName()) > 1 {
			name := notExistError.GetSpecifiedName()
			// Best-effort: attribute the unknown flag to the config file it came
			// from, if any (§8/05). Never blocks or alters the error itself.
			if src := attributeConfigFlag(name); src != "" {
				_, _ = fmt.Fprintf(os.Stderr, "in %s: unknown flag --%s\n", src, name)
			}
			printFlagSuggestion(flags, name)
		}
		os.Exit(1)
	}
}
