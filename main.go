package main

import (
	"github.com/boyter/scc/processor"
	"github.com/urfave/cli"
	"os"
	"runtime"
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
			Name:        "more, m",
			Usage:       "Set to check produce more output such as code vs complexity ranking",
			Destination: &processor.More,
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
			Name:        "flqs",
			Usage:       "Set the size of the file list queue",
			Value:       runtime.NumCPU(),
			Destination: &processor.FileListQueueSize,
		},
		cli.IntFlag{
			Name:        "frqs",
			Usage:       "Set the size of the file read content queue",
			Value:       runtime.NumCPU(),
			Destination: &processor.FileReadContentJobQueueSize,
		},
		cli.IntFlag{
			Name:        "fpqs",
			Usage:       "Set the size of the file process content queue",
			Value:       runtime.NumCPU(),
			Destination: &processor.FileProcessJobQueueSize,
		},
		cli.IntFlag{
			Name:        "fsqs",
			Usage:       "Set the size of the file summary queue",
			Value:       runtime.NumCPU() * 100,
			Destination: &processor.FileSummaryJobQueueSize,
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
