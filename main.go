package main

import (
	"github.com/boyter/scc/processor"
	"github.com/urfave/cli"
	"os"
	// "runtime/pprof"
)

//go:generate go run scripts/include.go
func main() {

	// f, _ := os.Create("scc.pprof")
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "scc"
	app.Version = "1.0.0"
	app.Usage = "Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation."
	app.UsageText = "scc DIRECTORY"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "pathblacklist, pbl",
			Usage:       "Which directories should be ignored as comma seperated list",
			Value:       ".git,.hg,.svn",
			Destination: &processor.PathBlacklist,
		},
		cli.StringFlag{
			Name:        "sort, s",
			Usage:       "Sort languages / files based on column [possible values: files, name, lines, blanks, code, comments, complexity]",
			Value:       "files",
			Destination: &processor.SortBy,
		},
		cli.StringFlag{
			Name:        "whitelist, wl",
			Usage:       "Restrict file extensions to just those provided as a comma seperated list E.G. go,java,js",
			Value:       "",
			Destination: &processor.WhiteListExtensions,
		},
		cli.BoolFlag{
			Name:        "files",
			Usage:       "Set to specify you want to see the output for every file",
			Destination: &processor.Files,
		},
		cli.BoolFlag{
			Name:        "verbose, v",
			Usage:       "Set to enable verbose output",
			Destination: &processor.Verbose,
		},
		cli.BoolFlag{
			Name:        "duplicates, d",
			Usage:       "Set to check for and remove duplicate files from stats and output",
			Destination: &processor.Duplicates,
		},
		cli.BoolFlag{
			Name:        "complexity, c",
			Usage:       "Set to skip complexity calculations note will be overridden if wide is set",
			Destination: &processor.Complexity,
		},
		cli.BoolFlag{
			Name:        "wide, w",
			Usage:       "Set to check produce more output such as complexity and code vs complexity ranking",
			Destination: &processor.More,
		},
		cli.Int64Flag{
			Name:        "averagewage, aw",
			Usage:       "Set as integer to set the average wage used for basic COCOMO calculation",
			Destination: &processor.AverageWage,
			Value:       56286,
		},
		cli.BoolFlag{
			Name:        "cocomo, co",
			Usage:       "Set to check remove cocomo calculation output",
			Destination: &processor.Cocomo,
		},
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Set to enable debug output",
			Destination: &processor.Debug,
		},
		cli.BoolFlag{
			Name:        "trace",
			Usage:       "Set to enable trace output, not reccomended for multiple files",
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
		processor.Process()
		return nil
	}

	app.Run(os.Args)
}
