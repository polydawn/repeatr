package cli

import (
	"github.com/codegangsta/cli"
	"io"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor/dispatch"
	"polydawn.net/repeatr/scheduler/dispatch"
)

func Main(args []string, journal io.Writer) {
	App := cli.NewApp()

	App.Name = "repeatr"
	App.Usage = "Run it. Run it again."
	App.Version = "0.0.1"

	App.Writer = journal

	bat := cli.StringSlice([]string{})

	App.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Run a formula",
			Action: func(c *cli.Context) { Run(c, journal) },
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "executor, e",
					Value: "chroot",
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

	App.Run(args)
}

func Run(c *cli.Context, journal io.Writer) {
	executor := executordispatch.Get(c.String("executor"))
	scheduler := schedulerdispatch.Get(c.String("scheduler"))
	paths := c.StringSlice("input")

	var formulae []def.Formula
	for _, path := range paths {
		formulae = append(formulae, LoadFormulaFromFile(path))
	}

	RunFormulae(scheduler, executor, journal, formulae...)
}
