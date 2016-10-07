package main

import (
	"fmt"
	"io"
	"os"

	"github.com/codegangsta/cli"

	"go.polydawn.net/repeatr/cmd/repeatr/cfg"
	rcli "go.polydawn.net/repeatr/core/cli"
)

func main() {
	os.Exit(Main(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

const (
	EXIT_SUCCESS      = 0
	EXIT_BADARGS      = 1
	EXIT_UNKNOWNPANIC = 2  // same code as golang uses when the process dies naturally on an unhandled panic.
	EXIT_JOB          = 10 // used to indicate a job reported a nonzero exit code (from cli commands that execute a single job).
	EXIT_USER         = 3  // grab bag for general user input errors (try to make a more specific code if possible/useful)
)

func Main(
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) (exitcode int) {
	subcommandHelpThunk := func(ctx *cli.Context) error {
		exitcode = EXIT_BADARGS
		if ctx.Args().Present() {
			cli.ShowCommandHelp(ctx, ctx.Args().First())
		} else {
			cli.ShowCommandHelp(ctx, "")
		}
		return nil
	}
	app := &cli.App{
		Name:      "repeatr",
		Usage:     "Run it. Run it again.",
		UsageText: "Repeatr runs processes in containers, provisioning their inputs and saving their outputs using reliable, immutable, content-addressable goodness.",
		Version:   "v0.13+dev",
		Writer:    stderr,
		Commands: []cli.Command{
			rcli.RunCommandPattern(stdout, stderr),
			rcli.TwerkCommandPattern(stdin, stdout, stderr),
			rcli.UnpackCommandPattern(stderr),
			rcli.ScanCommandPattern(stdout, stderr),
			rcli.ExploreCommandPattern(stdout, stderr),
			{
				Name:   "cfg",
				Usage:  "Manipulate config and formulas programmatically (parse, validate, etc).",
				Action: subcommandHelpThunk,
				Subcommands: []cli.Command{{
					Name:   "parse",
					Usage:  "Parse config and re-emit as json; error if any gross syntatic failures.",
					Action: cfgCmd.Parse(stdin, stdout, stderr),
				}},
			},
			{
				Name:  "version",
				Usage: "Shows the version of repeatr",
				Action: func(ctx *cli.Context) {
					cli.ShowVersion(ctx)
				},
			},
		},
		CommandNotFound: func(ctx *cli.Context, command string) {
			exitcode = EXIT_BADARGS
			fmt.Fprintf(stderr, "Incorrect usage: '%s' is not a %s subcommand\n", command, ctx.App.Name)
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Present() {
				cli.ShowCommandHelp(ctx, ctx.Args().First())
			} else {
				cli.ShowAppHelp(ctx)
			}
			return nil
		},
	}
	cli.VersionPrinter = func(ctx *cli.Context) {
		// Put some more info in our version printer.
		// Also, version goes to stdout.
		fmt.Fprintf(os.Stdout, "%v %v\n", ctx.App.Name, ctx.App.Version)
		fmt.Fprintf(os.Stdout, "git commit %v\n", rcli.GITCOMMIT)
		fmt.Fprintf(os.Stdout, "build date %v\n", rcli.BUILDDATE)
	}
	if err := app.Run(args); err != nil {
		exitcode = EXIT_BADARGS
		fmt.Fprintf(stderr, "Incorrect usage: %s", err)
	}
	return
}
