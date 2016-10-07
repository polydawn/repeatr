package examineCmd

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/core/executor/util"
	"go.polydawn.net/repeatr/lib/fshash"
	"go.polydawn.net/repeatr/lib/treewalk"
	"go.polydawn.net/repeatr/rio"
	"go.polydawn.net/repeatr/rio/filter"
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

func examinePath(thePath string, stdout io.Writer) {
	// Scan the whole arena contents back into a bucket of hashes and metadata.
	//  (If warehouses exposed their Buckets, that'd be handy.  But of course, not everyone uses those, so.)
	bucket := &fshash.MemoryBucket{}
	hasherFactory := sha512.New384
	filterset := filter.FilterSet{}
	if err := fshash.FillBucket(thePath, "", bucket, filterset, hasherFactory); err != nil {
		panic(err)
	}

	// Emit TDV.  (We'll quote&escape filenames so null-terminated lines aren't necessary -- this is meant for human consumption after all.)
	// Treewalk to the rescue, again.
	preVisit := func(node treewalk.Node) error {
		record := node.(fshash.RecordIterator).Record()
		m := record.Metadata
		// compute optional values
		var freehandValues []string
		if m.Linkname != "" {
			freehandValues = append(freehandValues, fmt.Sprintf("link:%q", m.Linkname))
		}
		if m.Typeflag == tar.TypeBlock || m.Typeflag == tar.TypeChar {
			freehandValues = append(freehandValues, fmt.Sprintf("major:%d", m.Devmajor))
			freehandValues = append(freehandValues, fmt.Sprintf("minor:%d", m.Devminor))
		} else if m.Typeflag == tar.TypeReg {
			freehandValues = append(freehandValues, fmt.Sprintf("hash:%s", base64.URLEncoding.EncodeToString(record.ContentHash)))
			freehandValues = append(freehandValues, fmt.Sprintf("len:%d", m.Size))
		}
		xattrsLen := len(m.Xattrs)
		if xattrsLen > 0 {
			sorted := make([]string, 0, xattrsLen)
			for k, v := range m.Xattrs {
				sorted = append(sorted, fmt.Sprintf("%q:%q", k, v))
			}
			sort.Strings(sorted)
			freehandValues = append(freehandValues, fmt.Sprintf("xattrs:[%s]", strings.Join(sorted, ",")))
		}
		// plug and chug
		fmt.Fprintf(stdout,
			"%q\t%c\t%#o\t%d\t%d\t%s\t%s\n",
			m.Name,
			m.Typeflag,
			m.Mode&07777,
			m.Uid,
			m.Gid,
			m.ModTime.UTC(),
			strings.Join(freehandValues, ","),
		)
		return nil
	}
	if err := treewalk.Walk(bucket.Iterator(), preVisit, nil); err != nil {
		panic(err)
	}
}
