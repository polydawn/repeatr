package cli

import (
	"fmt"
	"io"

	"github.com/codegangsta/cli"
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor/dispatch"
	"polydawn.net/repeatr/lib/guid"
)

func TwerkCommandPattern(stdin io.Reader, stdout, stderr io.Writer) cli.Command {
	return cli.Command{
		Name:  "twerk",
		Usage: "Run one-time-use interactive (thus nonrepeatable!) command.  All the defaults are filled in for you.  Great for experimentation.",
		Action: func(ctx *cli.Context) {
			executor := executordispatch.Get("runc")
			formula := def.Formula{
				Inputs: def.InputGroup{"main": {
					Type:      "tar",
					MountPath: "/",
					Hash:      "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT",
					Warehouses: []string{
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

			// TODO bonus points if you eventually can get the default mode to have no setuid binaries, in addition to making a spare user and dropping privs immediately.

			job := executor.Start(formula, def.JobID(guid.New()), stdin, ctx.App.Writer)
			go io.Copy(stdout, job.Outputs().Reader(1))
			go io.Copy(stderr, job.Outputs().Reader(2))
			result := job.Wait()
			if result.Error != nil {
				panic(Exit.NewWith(
					fmt.Sprintf("job execution errored: %s", result.Error.Message()),
					SetExitCode(EXIT_USER), // TODO review exit code
				))
			}
			if result.ExitCode != 0 {
				panic(Exit.NewWith(
					fmt.Sprintf("done; action finished with exit status %d", result.ExitCode),
					SetExitCode(EXIT_JOB),
				))
			}
			panic(Exit.NewWith("done; action reported successful exit status", SetExitCode(EXIT_SUCCESS)))
		},
	}
}
