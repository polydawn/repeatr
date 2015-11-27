package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/codegangsta/cli"
)

//go:generate ./version.go.tmpl

func Main(args []string, journal, output io.Writer) {
	App := cli.NewApp()

	App.Name = "repeatr"
	App.Usage = "Run it. Run it again."
	App.Version = "0.0.1"

	App.Writer = journal

	App.Commands = []cli.Command{
		RunCommandPattern(output),
		TwerkCommandPattern(os.Stdin, output, output), // FIXME this is too much loss of precision already
		UnpackCommandPattern(journal),
		ScanCommandPattern(output, journal),
		ExploreCommandPattern(output, journal),
	}

	// Reporting "no help topic for 'zyx'" and exiting with a *zero* is... silly.
	// A failure to hit a command should be an error.  Like, if a bash script does `repeatr somethingimportant`, there's no way it shouldn't *stop* when that's not there.
	App.CommandNotFound = func(ctx *cli.Context, command string) {
		fmt.Fprintf(ctx.App.Writer, "'%s %v' is not a repeatr subcommand\n", ctx.App.Name, command)
		os.Exit(int(EXIT_BADARGS))
	}

	// Put some more info in our version printer.
	// Global var.  Womp womp.
	// Also, version goes to stdout.
	cli.VersionPrinter = func(ctx *cli.Context) {
		fmt.Fprintf(os.Stdout, "%v v%v\n", ctx.App.Name, ctx.App.Version)
		fmt.Fprintf(os.Stdout, "git commit %v\n", GITCOMMIT)
		fmt.Fprintf(os.Stdout, "build date %v\n", BUILDDATE)
	}

	// Invoking version as a subcommand should also fly.
	App.Commands = append(App.Commands,
		cli.Command{
			Name:  "version",
			Usage: "Shows the version of repeatr",
			Action: func(ctx *cli.Context) {
				cli.ShowVersion(ctx)
			},
		},
	)

	App.Run(args)
}
