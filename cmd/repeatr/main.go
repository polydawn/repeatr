package main

import (
	"context"
	"fmt"
	"io"
	"os"

	. "github.com/warpfork/go-errcat"
	"gopkg.in/alecthomas/kingpin.v2"

	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/repeatr/fmt"
	"go.polydawn.net/repeatr/config"
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

type format string

const (
	format_Ansi = "ansi"
	format_Json = "json"
)

func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) behavior {
	// CLI boilerplate.
	app := kingpin.New("repeatr", "Functional computation.")
	app.HelpFlag.Short('h')
	app.UsageWriter(stderr)
	app.ErrorWriter(stderr)

	// Args struct defs and flag declarations.
	baseArgs := struct {
		Format string
	}{}
	app.Flag("format", "Output api format").
		Default(format_Ansi).
		EnumVar(&baseArgs.Format,
			format_Ansi, format_Json)
	bhvs := map[string]behavior{}
	{
		cmdRun := app.Command("run", "Execute a formula.")
		argsRun := struct {
			FormulaPath string
			Executor    string
		}{}
		cmdRun.Arg("formula", "Path to formula file.").
			Required().
			StringVar(&argsRun.FormulaPath)
		cmdRun.Flag("executor", "Select an executor system to use").
			Default("runc").
			EnumVar(&argsRun.Executor,
				"runc", "gvisor", "chroot")
		bhvs[cmdRun.FullCommand()] = behavior{&argsRun, func() error {
			memoDir := config.GetRepeatrMemoPath()
			printer := setupPrinter(format(baseArgs.Format), stdout, stderr)
			return RunCmd(ctx, argsRun.Executor, argsRun.FormulaPath, printer, memoDir)
		}}
	}
	{
		cmdTwerk := app.Command("twerk", "Execute a formula *interactively*.")
		argsTwerk := struct {
			FormulaPath string
			Executor    string
		}{}
		cmdTwerk.Arg("formula", "Path to formula file.").
			Required().
			StringVar(&argsTwerk.FormulaPath)
		cmdTwerk.Flag("executor", "Select an executor system to use").
			Default("runc").
			EnumVar(&argsTwerk.Executor,
				"runc", "chroot")
		bhvs[cmdTwerk.FullCommand()] = behavior{&argsTwerk, func() error {
			return Twerk(ctx, argsTwerk.Executor, argsTwerk.FormulaPath, stdin, stdout, stderr)
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

func setupPrinter(format format, stdout, stderr io.Writer) repeatrfmt.Printer {
	switch format {
	case format_Ansi:
		return repeatrfmt.NewAnsiPrinter(stdout, stderr)
	case format_Json:
		return repeatrfmt.NewJsonPrinter(stdout)
	default:
		panic("unreachable")
	}
}
