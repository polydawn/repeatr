package cli

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

	"polydawn.net/repeatr/core/executor/util"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/filter"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/lib/treewalk"
)

func ExploreCommandPattern(stdout, stderr io.Writer) cli.Command {
	return cli.Command{
		Name:  "explore",
		Usage: "describe a ware and the metadata of its contents, or a filesystem",
		Description: strings.Join([]string{
			"`repeatr explore` produces a human-readable manifest of every file in the named item",
			"(either wares or local filesystems may be examined), their properties, and their hashes.",
			"\n\n  ",
			"Output is structed as tab-delimited values -- you may feed it to an external `diff` program",
			"to compare one item with another; or, for easier reading, try piping it to `column -t`",
		}, " "),
		Subcommands: []cli.Command{
			{
				Name:  "ware",
				Usage: "explore a ware from a warehouse",
				Flags: []cli.Flag{
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
					if ctx.String("hash") == "" {
						panic(Error.New("%q is a required parameter", "hash"))
					}
					if ctx.String("where") == "" {
						panic(Error.New("%q is a required parameter", "where"))
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
						panic(Error.NewWith(err.Message(), SetExitCode(EXIT_BADARGS)))
					}).Catch(rio.WarehouseUnavailableError, func(err *errors.Error) {
						panic(Error.New("%s", err.Message()))
					}).Catch(rio.DataDNE, func(err *errors.Error) {
						panic(Error.New("%s", err.Message()))
					}).Catch(rio.HashMismatchError, func(err *errors.Error) {
						panic(Error.New("%s", err.Message()))
					}).Done()
				},
			},
			{
				Name:  "file",
				Usage: "explore a local filesystem",
				Action: func(ctx *cli.Context) {
					trailing := ctx.Args()
					switch len(trailing) {
					case 0:
						panic(Error.NewWith("`repeatr explore file` requires a path to explore", SetExitCode(EXIT_BADARGS)))
					case 1:
						break
					default:
						panic(Error.NewWith("`repeatr explore file` can only take one path at a time", SetExitCode(EXIT_BADARGS)))

					}
					// Check if it exists first for polite error message
					_, err := os.Lstat(trailing[0])
					if os.IsNotExist(err) {
						panic(Error.New("that path does not exist"))
					}
					// Examine the stuff.
					examinePath(trailing[0], stdout)
				},
			},
			// Other kinds of explore sub-command that may come later:
			//   - repeatr explore diff [item1] [item2]
			//        ... see, it's right about here that i start losing it in terms of how we should express object getting.  a bunch of flags named "--hash1" and "--hash2" are the obvious answer, but i pine for a more pleasant mechanism.
			//   - repeatr explore run [formula] [--output=name]
			//        As per `repeatr explore item`, but runs the formula and then immediately explores its output.
			//   - repeatr explore repeat [formula]
			//        Runs the given formula twice.  Checks that all the conjectured outputs are the same, exiting with a 1 and exploring diffs if they exist; exiting 0 if no diffs.
		},
	}
}

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
