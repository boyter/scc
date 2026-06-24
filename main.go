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

	// Handle --mcp flag before cobra to avoid interfering with stdio. Match both
	// the bare boolean form and the explicit --mcp=true form pflag accepts, so the
	// server starts consistently however the flag is spelled.
	if slices.ContainsFunc(os.Args[1:], func(a string) bool {
		return a == "--mcp" || a == "--mcp=true"
	}) {
		startMCPServer()
		return
	}

	// handle "scc @flags.txt" syntax. The sole-argument trigger is preserved;
	// only the splitter is swapped for the shared tokenizer, which adds comment
	// stripping, quote-aware tokenization and drops the old blank-line empty-arg
	// bug.
	if len(os.Args) == 2 && strings.HasPrefix(os.Args[1], "@") {
		filename := strings.TrimPrefix(os.Args[1], "@")
		b, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading flags from a file: %s\n", err)
			os.Exit(1)
		}
		os.Args = append([]string{os.Args[0]}, parseConfigArgs(string(b), true)...)
	}

	// What the user actually specified
	genuineCLI := slices.Clone(os.Args[1:])

	// Cobra's completion machinery must see the genuine argv. The shell invokes
	// the hidden __complete / __completeNoDesc commands on every TAB, and the
	// user-facing `completion` command generates the scripts; both key on the
	// subcommand sitting in args[0]. Prepending discovered config tokens would
	// shift it and break dynamic completion in any directory containing a ./.scc.
	// Config never influences completion output anyway, so skip discovery for it.
	var globalTokens, projectTokens []string
	if !isCompletionInvocation(os.Args) {
		noConfig, findRoot, explicitPath := preScanConfig(os.Args)
		var discoverErr error
		globalTokens, projectTokens, discoverErr = discoverConfigArgs(noConfig, findRoot, explicitPath)
		if discoverErr != nil {
			processor.PrintError(discoverErr.Error())
			os.Exit(1)
		}
	}

	// Fast path when no config was discovered: the merged list *is* the genuine
	// CLI, so bind write flags directly to the real vars and parse once, exactly
	// as scc does today. The discard / CLI-only split engages only when config is
	// actually present.
	// Ensures we follow old scc logic without config
	configPresent := len(globalTokens) != 0 || len(projectTokens) != 0

	// Build the merged argument list N.B. ORDER MATTERS HERE! CLI MUST COME LAST!
	var merged []string
	merged = append(merged, os.Args[0])
	merged = append(merged, globalTokens...)
	merged = append(merged, projectTokens...)
	merged = append(merged, genuineCLI...)

	// Write-flag bindings for the merged parse
	var discardOutput, discardReport, discardFormatMulti string
	bindings := &flagBindings{
		output:      &processor.FileOutput,
		report:      &processor.ReportOut,
		formatMulti: &processor.FormatMulti,
	}

	// if we use config, we NEVER allow writing to disk because someone could use that as
	// an attack vector, so we reset these options to ensure this is not a risk
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
    scc --report=out.html --report-title "myrepo" --report-skip cocomo

  Use a project config file (./.scc) or a global one (precedence: global < project < CLI):
    export SCC_CONFIG_PATH=~/.scc
    scc --config team.scc`,
		Version: processor.Version,
		Run: func(cmd *cobra.Command, args []string) {
			processor.DirFilePaths = args
			processor.ConfigureGc()
			processor.ConfigureLazy(true)

			// Detect if LOCOMO price/tps flags were explicitly set. Their default
			// is 0, which is ambiguous (unset vs. an explicit 0), so the processor
			// needs the "was it set?" bit to decide between the preset value and the
			// user override. Only pflag's Changed() knows this, hence here not there.
			processor.LocomoInputPriceSet = cmd.PersistentFlags().Changed("locomo-input-price")
			processor.LocomoOutputPriceSet = cmd.PersistentFlags().Changed("locomo-output-price")
			processor.LocomoTPSSet = cmd.PersistentFlags().Changed("locomo-tps")
			processor.LocomoCyclesSet = cmd.PersistentFlags().Changed("locomo-cycles")

			if v, err := cmd.PersistentFlags().GetBool("no-fold-authors"); err == nil && v {
				processor.FoldAuthors = false
			}

			// Source the write vars from the genuine CLI alone (file output is a
			// CLI-only capability), then warn if config tried to set one.
			if configPresent {
				cliSet := resolveWriteFlags(genuineCLI)
				warnIfConfigWrote(cmd.PersistentFlags(), cliSet)
			}

			// Merge the built-in defaults back into the empty-defaulted slice
			// flags, then flush any buffered config trace/debug.
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
	// in the working directory cannot shift args[1] and break completion.
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
			// Best-effort: attribute the unknown flag to the config file it came.
			if src := attributeConfigFlag(name); src != "" {
				_, _ = fmt.Fprintf(os.Stderr, "in %s: unknown flag --%s\n", src, name)
			}
			printFlagSuggestion(flags, name)
		}
		os.Exit(1)
	}
}
