package cli

import (
	"fmt"
	"io"

	"github.com/codegangsta/cli"
	"github.com/ugorji/go/codec"

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
				Value: "runc",
				Usage: "Which executor to use",
			},
			cli.BoolFlag{
				Name:  "ignore-job-exit",
				Usage: "If true, repeatr will exit with 0/success even if the job exited nonzero.",
			},
			cli.StringSliceFlag{
				Name:  "patch, p",
				Usage: "files with additional pieces of formula to apply before launch",
			},
		},
		Action: func(ctx *cli.Context) {
			// Parse args
			executor := executordispatch.Get(ctx.String("executor"))
			scheduler := schedulerdispatch.Get("linear")
			ignoreJobExit := ctx.Bool("ignore-job-exit")
			patchPaths := ctx.StringSlice("patch")
			// One (and only one) formula should follow;
			//  we don't have a way to unambiguously output more than one result formula at the moment.
			var formulaPath string
			switch l := len(ctx.Args()); {
			case l < 1:
				panic(Error.NewWith(
					"repeatr-run requires a path to a formula as the last argument",
					SetExitCode(EXIT_BADARGS),
				))
			case l > 1:
				panic(Error.NewWith(
					"repeatr-run requires exactly one formula as the last argument",
					SetExitCode(EXIT_BADARGS),
				))
			case l == 1:
				formulaPath = ctx.Args()[0]
			}
			// Parse formula
			formula := LoadFormulaFromFile(formulaPath)
			// Parse patches into formulas as well.
			//  Apply each one as it's loaded.
			for _, patchPath := range patchPaths {
				formula.ApplyPatch(LoadFormulaFromFile(patchPath))
			}

			// TODO Don't reeeeally want the 'run once' command going through the schedulers.
			//  Having a path that doesn't invoke that complexity unnecessarily, and also is more clearly allowed to use the current terminal, is want.

			// Invoke!
			result := RunFormula(scheduler, executor, formula, ctx.App.Writer)
			// Exit if the job failed collosally (if it just had a nonzero exit code, that's acceptable).
			if result.Error != nil {
				panic(Exit.NewWith(
					fmt.Sprintf("job execution errored: %s", result.Error.Message()),
					SetExitCode(EXIT_USER), // TODO review exit code
				))
			}

			// Output.
			// Join the results structure with the original formula, and emit the whole thing,
			//  just to keep it traversals consistent.
			// Note that all other logs, progress, terminals, etc are all routed to "journal" (typically, stderr),
			//  while this output is routed to "output" (typically, stdout), so it can be piped and parsed mechanically.
			formula.Outputs = result.Outputs
			err := codec.NewEncoder(output, &codec.JsonHandle{Indent: -1}).Encode(formula)
			if err != nil {
				panic(err)
			}
			output.Write([]byte{'\n'})
			// Exit nonzero with our own "your job did not report success" indicator code, if applicable.
			if result.ExitCode != 0 && !ignoreJobExit {
				panic(Exit.NewWith("job finished with non-zero exit status", SetExitCode(EXIT_JOB)))
			}
		},
	}
}
