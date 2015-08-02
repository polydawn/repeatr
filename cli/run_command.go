package cli

import (
	"io"

	"github.com/codegangsta/cli"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/dispatch"
	"polydawn.net/repeatr/scheduler"
	"polydawn.net/repeatr/scheduler/dispatch"
)

func RunCommandPattern(output io.Writer) cli.Command {
	bat := cli.StringSlice([]string{})

	return cli.Command{
		Name:  "run",
		Usage: "Run a formula",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "executor, e",
				Value: "chroot",
				Usage: "Which executor to use",
			},
			cli.StringFlag{
				Name:  "scheduler, s",
				Value: "linear",
				Usage: "Which scheduler to use",
			},
			cli.StringSliceFlag{
				Name:  "input, i",
				Value: &bat,
				Usage: "Location of input formulae (json format)",
			},
		},
		Action: func(c *cli.Context) {
			executor := executordispatch.Get(c.String("executor"))
			scheduler := schedulerdispatch.Get(c.String("scheduler"))
			formulaPaths := c.StringSlice("input")
			Run(executor, scheduler, formulaPaths, c.App.Writer)
		},
	}
}

func Run(executor executor.Executor, scheduler scheduler.Scheduler, formulaPaths []string, journal io.Writer) {
	var formulae []def.Formula
	for _, path := range formulaPaths {
		formulae = append(formulae, LoadFormulaFromFile(path))
	}

	// TODO Don't reeeeally want the 'run once' command going through the schedulers.
	//  Having a path that doesn't invoke that complexity unnecessarily, and also is more clearly allowed to use the current terminal, is want.

	if !RunFormulae(scheduler, executor, journal, formulae...) {
		panic(Error.NewWith("not all jobs completed successfully", SetExitCode(EXIT_USER)))
	}

	// output
	// TODO NEED JSON DESCRIPTION OF OUTPUTS TO BE EMITTED
}
