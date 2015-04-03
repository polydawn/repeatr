package cli

import (
	"github.com/codegangsta/cli"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor/dispatch"
	"polydawn.net/repeatr/scheduler/dispatch"
)

var App *cli.App

func init() {
	App = cli.NewApp()

	App.Name = "repeatr"
	App.Usage = "Run it. Run it again."
	App.Version = "0.0.1"

	bat := cli.StringSlice([]string{})

	App.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Run a formula",
			Action: Run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "executor, e",
					Value: "nsinit",
					Usage: "Which executor to use",
				},
				cli.StringFlag{
					Name:  "scheduler, s",
					Value: "linear",
					Usage: "Location of input formula (json format)",
				},
				cli.StringSliceFlag{
					Name:  "input, i",
					Value: &bat,
					Usage: "Location of input formulae (json format)",
				},
			},
		},
	}
}

func Run(c *cli.Context) {
	executor := executordispatch.Get(c.String("executor"))
	scheduler := schedulerdispatch.Get(c.String("scheduler"))
	paths := c.StringSlice("input")

	var formulae []def.Formula
	for _, path := range paths {
		formulae = append(formulae, LoadFormulaFromFile(path))
	}

	RunFormulae(scheduler, executor, formulae...)
}
