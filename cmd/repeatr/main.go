package main

import (
	"context"
	"fmt"
	"io"
	"os"

	. "github.com/polydawn/go-errcat"
	"gopkg.in/alecthomas/kingpin.v2"

	"go.polydawn.net/go-timeless-api/repeatr"
)

func main() {
	ctx := context.Background()
	bhv := Main(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr)
	err := bhv.action()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	exitCode := repeatr.ExitCodeForError(err)
	os.Exit(exitCode)
}

// Holder type which makes it easier for us to inspect
//  the args parser result in test code before running logic.
type behavior struct {
	parsedArgs interface{}
	action     func() error
}

func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) behavior {
	// CLI boilerplate.
	app := kingpin.New("repeatr", "Functional computation.")
	app.HelpFlag.Short('h')
	app.UsageWriter(stderr)
	app.ErrorWriter(stderr)

	// Args struct defs and flag declarations.
	bhvs := map[string]behavior{}
	{
		cmdRun := app.Command("run", "Execute a formula.")
		argsRun := struct {
			FormulaPath string
		}{}
		cmdRun.Arg("formula", "Path to formula file.").
			Required().
			StringVar(&argsRun.FormulaPath)
		bhvs[cmdRun.FullCommand()] = behavior{&argsRun, func() error {
			return Run(ctx, "chroot", argsRun.FormulaPath, stdout, stderr)
		}}
	}
	{
		cmdTwerk := app.Command("twerk", "Execute a formula *interactively*.")
		argsTwerk := struct {
			FormulaPath string
		}{}
		cmdTwerk.Arg("formula", "Path to formula file.").
			Required().
			StringVar(&argsTwerk.FormulaPath)
		bhvs[cmdTwerk.FullCommand()] = behavior{&argsTwerk, func() error {
			return Twerk(ctx, "chroot", argsTwerk.FormulaPath, stdin, stdout, stderr)
		}}
	}

	// Parse!
	parsedCmdStr, err := app.Parse(args[1:])
	if err != nil {
		return behavior{
			parsedArgs: err,
			action: func() error {
				return Errorf(repeatr.ErrUsage, "error parsing args: %s", err)
			},
		}
	}
	// Return behavior named by the command and subcommand strings.
	if bhv, ok := bhvs[parsedCmdStr]; ok {
		return bhv
	}
	panic("unreachable, cli parser must error on unknown commands")
}
