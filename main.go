package main

import (
	"github.com/boyter/scc/processor"
	"github.com/urfave/cli"
	"os"
)

//go:generate go run scripts/include.go
func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "scc"
	app.Version = "0.0.1"
	app.Usage = "Count lines of code in a directory"
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
			Name:        "format, f",
			Usage:       "What format output should be used [possible values: tabular, json, csv]",
			Value:       "tabular",
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
			Name:        "debug",
			Usage:       "Set to enable debug output",
			Destination: &processor.Debug,
		},
		cli.BoolFlag{
			Name:        "trace",
			Usage:       "Set to enable trace output, not reccomended for multiple files",
			Destination: &processor.Trace,
		},
		cli.IntFlag{
			Name:        "threads, j",
			Usage:       "Set the approx number of goroutines to use",
			Destination: &processor.NoThreads,
		},
		cli.BoolFlag{
			Name:        "garbagecollect, gc",
			Usage:       "Set to enable garbage collection during file walk. This may be required for very large directories",
			Destination: &processor.GarbageCollect,
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
