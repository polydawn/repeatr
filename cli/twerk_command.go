package cli

import (
	"fmt"
	"io"

	"github.com/codegangsta/cli"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor/dispatch"
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
					Hash:      "lzcqJKln2_H4TIoizNBCr0qoh8u_Nb_LRwARTZL2RumfbChX031pVl46dcSCG4q3",
					Warehouses: []string{
						"http+ca://repeatr.s3.amazonaws.com/assets/",
					},
				}},
				Action: def.Action{
					Entrypoint: []string{"bash"},
				},
			}

			// TODO bonus points if you eventually can get the default mode to have no setuid binaries, in addition to making a spare user and dropping privs immediately.

			job := executor.Start(formula, def.JobID(guid.New()), stdin, ctx.App.Writer)
			go io.Copy(stdout, job.Outputs().Reader(1))
			go io.Copy(stderr, job.Outputs().Reader(2))
			result := job.Wait()
			if result.Error != nil {
				fmt.Fprintf(ctx.App.Writer, "error: %s\n", result.Error)
			} else {
				fmt.Fprintf(ctx.App.Writer, "done; exit code %d\n", result.ExitCode)
			}
		},
	}
}
