package unpackCmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/core/executor/util"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rio"
	"go.polydawn.net/repeatr/rio/placer"
)

func Unpack(stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		if ctx.String("kind") == "" {
			panic(cmdbhv.ErrMissingParameter("kind"))
		}
		hash := ctx.String("hash")
		if hash == "" {
			panic(cmdbhv.ErrMissingParameter("hash"))
		}
		if ctx.String("where") == "" {
			panic(cmdbhv.ErrMissingParameter("where"))
		}
		placePath := ctx.String("place")
		if placePath == "" {
			placePath = fmt.Sprintf("./unpack-%s", hash)
		}

		log := log15.New()
		log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))

		if ctx.Bool("skip-exists") {
			if _, err := os.Stat(placePath); err == nil {
				return nil
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
			panic(&cmdbhv.ErrExit{err.Message(), cmdbhv.EXIT_BADARGS})
		}).Catch(rio.WarehouseUnavailableError, func(err *errors.Error) {
			panic(&cmdbhv.ErrExit{err.Message(), cmdbhv.EXIT_USER})
		}).Catch(rio.DataDNE, func(err *errors.Error) {
			panic(&cmdbhv.ErrExit{err.Message(), cmdbhv.EXIT_USER})
		}).Done()
		return nil
	}
}
