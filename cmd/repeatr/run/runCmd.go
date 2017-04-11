package runCmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/ugorji/go/codec"
	"github.com/urfave/cli"
	"go.polydawn.net/go-sup"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/act/remote/server"
	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/api/hitch"
	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/core/actors/runner"
	"go.polydawn.net/repeatr/core/actors/terminal"
	"go.polydawn.net/repeatr/core/executor/dispatch"
)

func Run(stdout, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
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
			panic(meep.Meep(&cmdbhv.ErrBadArgs{
				Message: "`repeatr run` requires a path to a formula as the last argument",
			}))
		case l > 1:
			panic(meep.Meep(&cmdbhv.ErrBadArgs{
				Message: "`repeatr run` requires exactly one formula as the last argument",
			}))
		case l == 1:
			formulaPath = ctx.Args()[0]
		}
		// Parse formula
		formula := hitch.LoadFormulaFromFile(formulaPath)
		// Parse patches into formulas as well.
		//  Apply each one as it's loaded.
		for _, patchPath := range patchPaths {
			formula.ApplyPatch(*hitch.LoadFormulaFromFile(patchPath))
		}
		// Any env var overrides stomp even on top of patches.
		for _, envArg := range envArgs {
			parts := strings.SplitN(envArg, "=", 2)
			if len(parts) < 2 {
				panic(meep.Meep(&cmdbhv.ErrBadArgs{
					Message: "env arguments must have an equal sign (like this: '-e KEY=val').",
				}))
			}
			formula.ApplyPatch(def.Formula{Action: def.Action{
				Env: map[string]string{parts[0]: parts[1]},
			}})
		}

		// Create a local formula runner, and power it with a supervisor.
		runner := runner.New(runner.Config{
			Executor: executor,
		})
		go sup.NewTask().Run(runner.Run)

		// Request run.
		runID := runner.StartRun(formula)

		// Park our routine... in one of two radically different ways:
		//  - either following events and proxying them to terminal,
		//  - or following events, serializing them, and
		//    shunting them out API-like!
		if serialize {
			// Rig a publisher and set it to fly straight on til sunrise.
			publisher := &server.RunObserverPublisher{
				Proxy:           runner,
				RunID:           runID,
				Output:          stdout,
				RecordSeparator: []byte{'\n'},
				Codec:           &codec.JsonHandle{},
			}
			writ := sup.NewTask()
			writ.Run(publisher.Run)
			if err := writ.Err(); err != nil {
				// This should almost never be hit, because we push
				// most interesting errors out in the `runRecord.Failure` field.
				// But some really dire things do leave through the window:
				// for example if our serializer config was bunkum, we
				// really have no choice but to crash hard here and
				// simply try to make it visible.
				panic(err)
			}
			// We always exitcode as success in API mode!
			//  We don't care whether there was a huge error in the
			//  `runRecord.Failure` field -- if so, it was reported
			//  through the serial API stream like everything else.
			return nil
		}
		// Else: Okay, human/terminal mode it is!
		runRecord := terminal.Consume(runner, runID, stderr)

		// Raise the error that got in the way of execution, if any.
		cmdbhv.TryPlanToExit.MustHandle(runRecord.Failure)

		// Output the results structure.
		//  This goes on stdout (everything is stderr) and so should be parsable.
		//  We strip some fields that aren't very useful to single-task manual runs.
		runRecord.HID = ""
		runRecord.FormulaHID = ""
		if err := codec.NewEncoder(stdout, &codec.JsonHandle{Indent: -1}).Encode(runRecord); err != nil {
			panic(meep.Meep(
				&meep.ErrProgrammer{},
				meep.Cause(fmt.Errorf("Transcription error: %s", err)),
			))
		}
		stdout.Write([]byte{'\n'})
		// Exit nonzero with our own "your job did not report success" indicator code, if applicable.
		exitCode := runRecord.Results["$exitcode"].Hash
		if exitCode != "0" && !ignoreJobExit {
			panic(&cmdbhv.ErrExit{
				Message: fmt.Sprintf("job finished with non-zero exit status %s", exitCode),
				Code:    cmdbhv.EXIT_JOB,
			})
		}
		return nil
	}
}
