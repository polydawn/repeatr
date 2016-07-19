package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/actors"
	"polydawn.net/repeatr/core/executor/dispatch"
)

func TwerkCommandPattern(stdin io.Reader, stdout, stderr io.Writer) cli.Command {
	return cli.Command{
		Name:  "twerk",
		Usage: "Run one-time-use interactive (thus nonrepeatable!) command.  All the defaults are filled in for you.  Great for experimentation.",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "executor",
				Value: "runc",
				Usage: "Which executor to use. Using anything other than runc at this time is not particularly useful or functional.",
			},
			cli.StringSliceFlag{
				Name:  "patch, p",
				Usage: "files with additional pieces of formula to apply before launch",
			},
			cli.StringSliceFlag{
				Name:  "env, e",
				Usage: "apply additional environment vars to formula before launch (overrides 'patch').  Format like '-e KEY=val'",
			},
			cli.StringFlag{
				Name:  "policy, P",
				Value: string(def.PolicyRoutine),
				Usage: "Which capabilities policy to use",
			},
		},
		Action: func(ctx *cli.Context) {
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
				formula.ApplyPatch(LoadFormulaFromFile(patchPath))
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
				panic(Error.NewWith(buffer.String(), SetExitCode(EXIT_BADARGS)))
			}
			// Any policy overrides apply after patches
			formula.Action.Policy = def.Policy(policy)

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

			if result.Failure != nil {
				panic(Exit.NewWith(
					fmt.Sprintf("job execution errored: %s", result.Failure),
					SetExitCode(EXIT_USER), // TODO review exit code
				))
			}
			exitCode := result.Results["$exitcode"].Hash
			if exitCode != "0" {
				panic(Exit.NewWith(
					fmt.Sprintf("done; action finished with exit status %s", exitCode),
					SetExitCode(EXIT_JOB),
				))
			}
			panic(Exit.NewWith("done; action reported successful exit status", SetExitCode(EXIT_SUCCESS)))
		},
	}
}
