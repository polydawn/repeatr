package cli

import (
	"fmt"
	"io"

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
		Action: func(ctx *cli.Context) {
			execr := executordispatch.Get("runc")
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

			// Set up a logger.
			log := log15.New()
			log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))

			// Create a local formula runner, and fire.
			runner := actor.NewFormulaRunner(execr, log)
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
