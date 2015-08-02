package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/codegangsta/cli"
	"polydawn.net/repeatr/executor/dispatch"
	"polydawn.net/repeatr/scheduler/dispatch"
)

func RunCommandPattern(output io.Writer) cli.Command {
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
			cli.StringFlag{
				Name:  "input, i",
				Usage: "Location of input formula (json format)",
			},
		},
		Action: func(ctx *cli.Context) {
			// Parse args
			executor := executordispatch.Get(ctx.String("executor"))
			scheduler := schedulerdispatch.Get(ctx.String("scheduler"))
			formulaPaths := ctx.String("input")
			// Parse formula
			formula := LoadFormulaFromFile(formulaPaths)

			// TODO Don't reeeeally want the 'run once' command going through the schedulers.
			//  Having a path that doesn't invoke that complexity unnecessarily, and also is more clearly allowed to use the current terminal, is want.

			// Invoke!
			result := RunFormula(scheduler, executor, formula, ctx.App.Writer)
			// Exit if the job failed collosally (if it just had a nonzero exit code, that's acceptable).
			if result.Error != nil {
				panic(Error.NewWith("job execution errored", SetExitCode(EXIT_USER)))
			}

			// Output.
			// Note that all other logs, progress, terminals, etc are all routed to "journal" (typically, stderr),
			//  while this output is routed to "output" (typically, stdout), so it can be piped and parsed mechanically.
			msg, err := json.Marshal(result.Outputs)
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(output, "%s\n", string(msg))
			// Exit nonzero with our own "your job did not report success" indicator code, if applicable.
			if result.ExitCode != 0 {
				panic(Exit.NewWith("job finished with non-zero exit status", SetExitCode(EXIT_JOB)))
			}
		},
	}
}
