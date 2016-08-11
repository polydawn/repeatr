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
	App.Version = "v0.13+dev"

	App.Writer = journal

	App.Commands = []cli.Command{
		RunCommandPattern(output),
		TwerkCommandPattern(os.Stdin, output, journal),
		UnpackCommandPattern(journal),
		ScanCommandPattern(output, journal),
		ExploreCommandPattern(output, journal),
		CfgCommandPattern(os.Stdin, output, journal),
	}

	// Slight touch to the phrasing on subcommands not found.
	App.CommandNotFound = func(ctx *cli.Context, command string) {
		panic(Exit.NewWith(
			fmt.Sprintf("Incorrect usage: '%s %v' is not a repeatr subcommand\n", ctx.App.Name, command),
			SetExitCode(EXIT_BADARGS),
		))
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

	if err := App.Run(args); err != nil {
		panic(Exit.NewWith(
			fmt.Sprintf("Incorrect usage: %s", err),
			SetExitCode(EXIT_BADARGS),
		))
	}
}
