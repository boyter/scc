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
			Usage:       "Which directories should be ignored as comma seperated list `.git,.hg,.svn`",
			Value:       ".git,.hg,.svn",
			Destination: &processor.PathBlacklist,
		},
		cli.BoolFlag{
			Name:        "files",
			Usage:       "Set this to anything non blank to specify you want to see the output for every file",
			Destination: &processor.Files,
		},
		cli.BoolFlag{
			Name:        "verbose, v",
			Usage:       "Set this to enable verbose output",
			Destination: &processor.Verbose,
		},
		cli.BoolFlag{
			Name:        "veryverbose, vv",
			Usage:       "Set this to enable very verbose output",
			Destination: &processor.VeryVerbose,
		},
	}

	// Override version flag
	cli.VersionFlag = cli.BoolFlag{
		Name:  "version, ver",
		Usage: "print the version",
	}

	app.Action = func(c *cli.Context) error {
		processor.DirFilePaths = c.Args()
		processor.Process()
		return nil
	}

	app.Run(os.Args)
}
