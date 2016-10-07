package twerkCmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/api/hitch"
	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/core/actors"
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

		// Set up a logger.
		log := log15.New()
		log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))

		// Create a local formula runner, and fire.
		runner := actor.NewFormulaRunner(executor, log)
		runner.InjectStdin(stdin)
		runID := runner.StartRun(&formula)

		// Stream job output to terminal in real time
		runner.FollowStreams(runID, stdout, stderr)

		// Wait for results.
		result := runner.FollowResults(runID)

		// Raise any errors that got in the way of execution.
		meep.TryPlan{
			// TODO this should filter out DataDNE, HashMismatch, etc.
			// examineCmd does a better job of this.
			// come back to this after more meep integration.
			{CatchAny: true,
				Handler: meep.TryHandlerMapto(&cmdbhv.ErrRunFailed{})},
		}.MustHandle(result.Failure)

		exitCode := result.Results["$exitcode"].Hash
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
