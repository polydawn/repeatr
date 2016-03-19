package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/core/executor/util"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/placer"
)

func UnpackCommandPattern(stderr io.Writer) cli.Command {
	return cli.Command{
		Name:  "unpack",
		Usage: "fetch a ware and unpack it to a local filesystem",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "place",
				Usage: "Optional.  Where to place the filesystem.  Defaults to \"./unpack-[hash]\"",
			},
			cli.StringFlag{
				Name:  "kind",
				Usage: "What kind of data storage format to work with.",
			},
			cli.StringFlag{
				Name:  "hash",
				Usage: "The ID of the object to explore.",
			},
			cli.StringFlag{
				Name:  "where",
				Usage: "A URL giving coordinates to a warehouse where repeatr should find the object to explore.",
			},
		},
		Action: func(ctx *cli.Context) {
			if ctx.String("kind") == "" {
				panic(Error.New("%q is a required parameter", "kind"))
			}
			hash := ctx.String("hash")
			if hash == "" {
				panic(Error.New("%q is a required parameter", "hash"))
			}
			if ctx.String("where") == "" {
				panic(Error.New("%q is a required parameter", "where"))
			}
			placePath := ctx.String("place")
			if placePath == "" {
				placePath = fmt.Sprintf("./unpack-%s", hash)
			}

			log := log15.New()
			log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))

			try.Do(func() {
				// Make the unpack location, if it doesn't exist.
				os.MkdirAll(placePath, 0755)
				// Materialize the things.
				arena := util.DefaultTransmat().Materialize(
					integrity.TransmatKind(ctx.String("kind")),
					integrity.CommitID(hash),
					[]integrity.SiloURI{integrity.SiloURI(ctx.String("where"))},
					log,
				)
				defer arena.Teardown()
				// Copy the materialized data into its permanent new home.
				//  This feels kind of redundant at first glance (e.g., "why couldn't
				//  we just materialize it in the right place the first time?"), but
				//  makes sense when you remember transmats might just be returning a
				//  pointer into a shared cache that already existed (or other non-relocatable
				//  excuse for a filesystem, e.g. fuse happened or something).
				placer.CopyingPlacer(arena.Path(), placePath, true, false)
			}).Catch(integrity.ConfigError, func(err *errors.Error) {
				panic(Error.NewWith(err.Message(), SetExitCode(EXIT_BADARGS)))
			}).Catch(integrity.WarehouseUnavailableError, func(err *errors.Error) {
				panic(Error.New("%s", err.Message()))
			}).Catch(integrity.DataDNE, func(err *errors.Error) {
				panic(Error.New("%s", err.Message()))
			}).Done()
		},
	}
}
