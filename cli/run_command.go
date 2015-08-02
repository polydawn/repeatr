package cli

import (
	"encoding/json"
	"fmt"
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
			Run(executor, scheduler, formulaPaths, c.App.Writer, output)
		},
	}
}

func Run(executor executor.Executor, scheduler scheduler.Scheduler, formulaPaths []string, journal io.Writer, output io.Writer) {
	var formulae []def.Formula
	for _, path := range formulaPaths {
		formulae = append(formulae, LoadFormulaFromFile(path))
	}

	// TODO Don't reeeeally want the 'run once' command going through the schedulers.
	//  Having a path that doesn't invoke that complexity unnecessarily, and also is more clearly allowed to use the current terminal, is want.
	// TODO / NOTE : yeah, this is still worryingly ambiguous:
	//  - Current behavior is to emit nothing for outputs on failed jobs, whic his... not unlikely to bork your shell scripts
	//  - Current behavior only returns non-zero exits on major failure to launch/monitor... not if jobs exit nonzero, which is likely to bork your shell scripts
	//  - If you actually *use* the multi-formula feature, your outputs are not going to be fun to re-demux.
	//  Increasingly it seems like this idea of multiple formulae through a single shell command should be removed outright.  The API of a single CLI command isn't up to it.

	// Prepare to collect results.
	results := make(chan def.JobResult)

	// Output... as we go, yes.
	// Note that all other logs, progress, terminals, etc are all routed to "journal" (typically, stderr),
	//  while this output is routed to "output" (typically, stdout), so it can be piped and parsed mechanically.
	go func() {
		// Sync note: `results` being unbuffered is critical to this being always run before the terminal return of RunFormulae.
		for result := range results {
			msg, err := json.Marshal(result.Outputs)
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(output, "%s\n", string(msg))
			// consider: should exit code maybe be added to def.Formula for record keeping...?
		}
	}()

	if !RunFormulae(scheduler, executor, journal, results, formulae...) {
		panic(Error.NewWith("not all jobs completed successfully", SetExitCode(EXIT_USER)))
	}
}
