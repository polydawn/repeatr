package twerkCmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/codegangsta/cli"
	"go.polydawn.net/go-sup"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/api/hitch"
	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/core/actors/runner"
	"go.polydawn.net/repeatr/core/actors/terminal"
	executordispatch "go.polydawn.net/repeatr/core/executor/dispatch"
)

func Twerk(stdin io.Reader, stdout, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		executor := executordispatch.Get(ctx.String("executor"))
		patchPaths := ctx.StringSlice("patch")
		envArgs := ctx.StringSlice("env")
		policy := ctx.String("policy")

		formula := def.Formula{
			Inputs: def.InputGroup{"main": {
				Type:      "tar",
				MountPath: "/",
				Hash:      "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT",
				Warehouses: def.WarehouseCoords{
					"http+ca://repeatr.s3.amazonaws.com/assets/",
				},
			}},
			Action: def.Action{
				Entrypoint: []string{"bash"},
				Escapes: def.Escapes{
					Mounts: []def.Mount{{
						SourcePath: ".",
						TargetPath: "/whee",
						Writable:   true,
					}},
				},
				Cwd: "/whee",
			},
		}

		// Parse patches into formulas as well.
		//  Apply each one as it's loaded.
		for _, patchPath := range patchPaths {
			formula.ApplyPatch(*hitch.LoadFormulaFromFile(patchPath))
		}

		if !func(policy string) bool {
			for _, policyType := range def.PolicyValues {
				if policy == string(policyType) {
					return true
				}
			}
			return false
		}(policy) {
			var buffer bytes.Buffer
			buffer.WriteString("Must select a valid policy:")
			for _, value := range def.PolicyValues {
				buffer.WriteString(" ")
				buffer.WriteString(string(value))
			}
			panic(meep.Meep(&cmdbhv.ErrBadArgs{
				Message: buffer.String(),
			}))
		}
		// Any policy overrides apply after patches
		formula.Action.Policy = def.Policy(policy)

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
			Stdin:    stdin,
		})
		go sup.NewTask().Run(runner.Run)

		// Request run.
		runID := runner.StartRun(&formula)

		// Park our routine, following events and proxying them to terminal.
		runRecord := terminal.Consume(runner, runID, stdout, stderr)

		// Raise any errors that got in the way of execution.
		meep.TryPlan{
			// TODO this should filter out DataDNE, HashMismatch, etc.
			// examineCmd does a better job of this.
			// come back to this after more meep integration.
			{CatchAny: true,
				Handler: meep.TryHandlerMapto(&cmdbhv.ErrRunFailed{})},
		}.MustHandle(runRecord.Failure)

		exitCode := runRecord.Results["$exitcode"].Hash
		if exitCode != "0" {
			panic(&cmdbhv.ErrExit{
				Message: fmt.Sprintf("done; action finished with exit status %s", exitCode),
				Code:    cmdbhv.EXIT_JOB,
			})
		}
		panic(&cmdbhv.ErrExit{
			Message: "done; action reported successful exit status",
			Code:    cmdbhv.EXIT_SUCCESS,
		})
	}
}
