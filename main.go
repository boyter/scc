package main

import (
	"github.com/boyter/scc/processor"
	"github.com/urfave/cli"
	"os"
)

//go:generate go run scripts/include.go
func main() {
	//f, _ := os.Create("scc.pprof")
	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "scc"
	app.Version = "1.10.0"
	app.Usage = "Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation."
	app.UsageText = "scc DIRECTORY"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "languages",
			Usage:       "Print out supported languages and extensions",
			Destination: &processor.Languages,
		},
		cli.StringFlag{
			Name:        "format, f",
			Usage:       "Set output format [possible values: tabular, wide, json, csv]",
			Destination: &processor.Format,
			Value:       "tabular",
		},
		cli.StringFlag{
			Name:        "output, o",
			Usage:       "Save to file, defaults to stdout",
			Destination: &processor.FileOutput,
		},
		cli.StringFlag{
			Name:        "pathblacklist, pbl",
			Usage:       "Which directories should be ignored as comma separated list",
			Value:       ".git,.hg,.svn",
			Destination: &processor.PathBlacklist,
		},
		cli.StringFlag{
			Name:        "sort, s",
			Usage:       "Sort based on column [possible values: files, name, lines, blanks, code, comments, complexity]",
			Value:       "files",
			Destination: &processor.SortBy,
		},
		cli.StringFlag{
			Name:        "exclude, e",
			Usage:       "Ignore files and directories matching supplied regular expression",
			Value:       "",
			Destination: &processor.Exclude,
		},
		cli.StringFlag{
			Name:        "whitelist, wl",
			Usage:       "Restrict file extensions to just those provided as a comma separated list E.G. go,java,js",
			Value:       "",
			Destination: &processor.WhiteListExtensions,
		},
		cli.BoolFlag{
			Name:        "files",
			Usage:       "Display output for every file",
			Destination: &processor.Files,
		},
		cli.BoolFlag{
			Name:        "verbose, v",
			Usage:       "Enable verbose output",
			Destination: &processor.Verbose,
		},
		cli.BoolFlag{
			Name:        "duplicates, d",
			Usage:       "Check for and remove duplicate files from stats and output",
			Destination: &processor.Duplicates,
		},
		cli.BoolFlag{
			Name:        "complexity, c",
			Usage:       "Skip complexity calculations, note this will be overridden if --wide -w is set",
			Destination: &processor.Complexity,
		},
		cli.BoolFlag{
			Name:        "wide, w",
			Usage:       "Wider output with additional statistics",
			Destination: &processor.More,
		},
		cli.Int64Flag{
			Name:        "averagewage, aw",
			Usage:       "Integer to override the average wage value used for basic COCOMO calculation",
			Destination: &processor.AverageWage,
			Value:       56286,
		},
		cli.BoolFlag{
			Name:        "cocomo, co",
			Usage:       "Set to check remove COCOMO calculation output",
			Destination: &processor.Cocomo,
		},
		cli.IntFlag{
			Name:        "filegccount, fgc",
			Usage:       "How many files to parse before turning the GC on",
			Destination: &processor.GcFileCount,
			Value:       10000,
		},
		cli.BoolFlag{
			Name:        "binary",
			Usage:       "Disable binary file detection",
			Destination: &processor.DisableCheckBinary,
		},
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Enable debug output",
			Destination: &processor.Debug,
		},
		cli.BoolFlag{
			Name:        "trace",
			Usage:       "Enable trace output, not recommended when processing multiple files",
			Destination: &processor.Trace,
		},
	}

	// Override the default version flag because we want v for verbose
	cli.VersionFlag = cli.BoolFlag{
		Name:  "version, ver",
		Usage: "Print the version",
	}

	app.Action = func(c *cli.Context) error {
		processor.DirFilePaths = c.Args()
		processor.ConfigureGc()
		processor.Process()
		return nil
	}

	app.Run(os.Args)
}
