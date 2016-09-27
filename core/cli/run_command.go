package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/ugorji/go/codec"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor/dispatch"
)

func RunCommandPattern(output io.Writer, journal io.Writer) cli.Command {
	return cli.Command{
		Name:  "run",
		Usage: "Run a formula",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "executor",
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
			cli.StringSliceFlag{
				Name:  "env, e",
				Usage: "apply additional environment vars to formula before launch (overrides 'patch').  Format like '-e KEY=val'",
			},
			cli.BoolFlag{
				Name:  "serialize, s",
				Usage: "serialize output onto stdout",
			},
		},
		Action: func(ctx *cli.Context) {
			// Parse args
			executor := executordispatch.Get(ctx.String("executor"))
			ignoreJobExit := ctx.Bool("ignore-job-exit")
			patchPaths := ctx.StringSlice("patch")
			envArgs := ctx.StringSlice("env")
			serialize := ctx.Bool("serialize")
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
			// Any env var overrides stomp even on top of patches.
			for _, envArg := range envArgs {
				parts := strings.SplitN(envArg, "=", 2)
				if len(parts) < 2 {
					panic(Error.NewWith(
						"env arguments must have an equal sign (like this: '-e KEY=val').",
						SetExitCode(EXIT_BADARGS),
					))
				}
				formula.ApplyPatch(def.Formula{Action: def.Action{
					Env: map[string]string{parts[0]: parts[1]},
				}})
			}

			// set up journal and logger based on flags
			log := log15.New()

			if serialize {
				// set up serializer for journal stream
				js := &journalSerializer{
					Delegate: output,
				}
				journal = js
				// use our custom logHandler to serialize results uniformly
				log.SetHandler(logHandler(output))
			} else {
				// no serialization of output, write directly to journal
				log.SetHandler(log15.StreamHandler(journal, log15.TerminalFormat()))
			}

			// Invoke!
			result := RunFormula(executor, formula, journal, log)
			// Exit if the job failed collosally (if it just had a nonzero exit code, that's acceptable).
			if result.Failure != nil {
				panic(Exit.NewWith(
					fmt.Sprintf("job execution errored: %s", result.Failure),
					SetExitCode(EXIT_USER), // TODO review exit code
				))
			}

			// Output the results structure.
			//  This goes on stdout (everything is stderr) and so should be parsable.
			//  We strip some fields that aren't very useful to single-task manual runs.
			result.HID = ""
			result.FormulaHID = ""
			var err error
			if serialize {
				err = serializeResult(output, "result", result)
			} else {
				err = codec.NewEncoder(output, &codec.JsonHandle{Indent: -1}).Encode(result)
			}
			if err != nil {
				panic(err)
			}
			output.Write([]byte{'\n'})
			// Exit nonzero with our own "your job did not report success" indicator code, if applicable.
			if result.Results["$exitcode"].Hash != "0" && !ignoreJobExit {
				panic(Exit.NewWith("job finished with non-zero exit status", SetExitCode(EXIT_JOB)))
			}
		},
	}
}
