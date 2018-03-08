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
	}

	app.Action = func(c *cli.Context) error {
		processor.Process()
		return nil
	}

	app.Run(os.Args)
}
