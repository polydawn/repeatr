package examineCmd

import (
	"io"
	"os"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/core/executor/util"
	"go.polydawn.net/repeatr/rio"
)

func ExamineWare(stdin io.Reader, stdout, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		if ctx.String("kind") == "" {
			panic(cmdbhv.ErrMissingParameter("kind"))
		}
		if ctx.String("hash") == "" {
			panic(cmdbhv.ErrMissingParameter("hash"))
		}
		if ctx.String("where") == "" {
			panic(cmdbhv.ErrMissingParameter("where"))
		}

		log := log15.New()
		log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))

		try.Do(func() {
			// Materialize the things.
			arena := util.DefaultTransmat().Materialize(
				rio.TransmatKind(ctx.String("kind")),
				rio.CommitID(ctx.String("hash")),
				[]rio.SiloURI{rio.SiloURI(ctx.String("where"))},
				log,
			)
			defer arena.Teardown()
			// Examine 'em.
			examinePath(arena.Path(), stdout)
		}).Catch(rio.ConfigError, func(err *errors.Error) {
			panic(&cmdbhv.ErrExit{err.Message(), cmdbhv.EXIT_BADARGS})
		}).Catch(rio.WarehouseUnavailableError, func(err *errors.Error) {
			panic(&cmdbhv.ErrExit{err.Message(), cmdbhv.EXIT_USER})
		}).Catch(rio.DataDNE, func(err *errors.Error) {
			panic(&cmdbhv.ErrExit{err.Message(), cmdbhv.EXIT_USER})
		}).Catch(rio.HashMismatchError, func(err *errors.Error) {
			panic(&cmdbhv.ErrExit{err.Message(), cmdbhv.EXIT_USER})
		}).Done()
		return nil
	}
}

func ExamineFile(stdin io.Reader, stdout, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		trailing := ctx.Args()
		switch len(trailing) {
		case 0:
			panic(meep.Meep(&cmdbhv.ErrBadArgs{
				Message: "`repeatr examine file` requires a path to examine"}))
		case 1:
			break
		default:
			panic(meep.Meep(&cmdbhv.ErrBadArgs{
				Message: "`repeatr examine file` can only take one path at a time"}))

		}
		// Check if it exists first for polite error message
		_, err := os.Lstat(trailing[0])
		if os.IsNotExist(err) {
			panic(meep.Meep(&cmdbhv.ErrBadArgs{
				Message: "that path does not exist"}))
		}
		// Examine the stuff.
		examinePath(trailing[0], stdout)
		return nil
	}
}

// Other kinds of examine sub-command that may come later:
//   - repeatr examine diff [item1] [item2]
//        ... see, it's right about here that i start losing it in terms of how we should express object getting.  a bunch of flags named "--hash1" and "--hash2" are the obvious answer, but i pine for a more pleasant mechanism.
//   - repeatr examine run [formula] [--output=name]
//        As per `repeatr examine item`, but runs the formula and then immediately explores its output.
//   - repeatr examine repeat [formula]
//        Runs the given formula twice.  Checks that all the conjectured outputs are the same, exiting with a 1 and exploring diffs if they exist; exiting 0 if no diffs.
