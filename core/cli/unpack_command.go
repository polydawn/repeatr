package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"go.polydawn.net/repeatr/core/executor/util"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rio"
	"go.polydawn.net/repeatr/rio/placer"
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
			cli.BoolFlag{
				Name: "skip-exists",
				Usage: "If a file already exists at at '--place=%s', assume it's correct and exit immediately.  If this flag is not provided, the default behavior is to do the whole unpack, rolling over the existing files." +
					"  BE WARY of using this: it's effectively caching with no cachebusting rule.  Caveat emptor.",
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

			if ctx.Bool("skip-exists") {
				if _, err := os.Stat(placePath); err == nil {
					return
				}
			}

			try.Do(func() {
				// Materialize the things.
				arena := util.DefaultTransmat().Materialize(
					rio.TransmatKind(ctx.String("kind")),
					rio.CommitID(hash),
					[]rio.SiloURI{rio.SiloURI(ctx.String("where"))},
					log,
				)
				defer arena.Teardown()
				// Pick a temp path we'll move things into first.
				//  The materialize arena can't be jumped into the final resting
				//  place atomically, so we get it close, then do an atomic op last.
				placePathDir := filepath.Dir(placePath)
				placePathName := filepath.Base(placePath)
				tmpPlacePath := filepath.Join(
					placePathDir,
					".tmp."+placePathName+"."+guid.New(),
				)
				// Copy the materialized data into (within-one-step-of) its permanent new home.
				//  This feels kind of redundant at first glance (e.g., "why couldn't
				//  we just materialize it in the right place the first time?"), but
				//  makes sense when you remember transmats might just be returning a
				//  pointer into a shared cache that already existed (or other non-relocatable
				//  excuse for a filesystem, e.g. fuse happened or something).
				placer.CopyingPlacer(arena.Path(), tmpPlacePath, true, false)
				// Atomic move into final place.
				if err := os.Rename(tmpPlacePath, placePath); err != nil {
					panic(err)
				}
			}).Catch(rio.ConfigError, func(err *errors.Error) {
				panic(Error.NewWith(err.Message(), SetExitCode(EXIT_BADARGS)))
			}).Catch(rio.WarehouseUnavailableError, func(err *errors.Error) {
				panic(Error.New("%s", err.Message()))
			}).Catch(rio.DataDNE, func(err *errors.Error) {
				panic(Error.New("%s", err.Message()))
			}).Done()
		},
	}
}
