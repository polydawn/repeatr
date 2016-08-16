package file

import (
	"crypto/sha512"
	"encoding/base64"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/flak"
	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/rio"
	"go.polydawn.net/repeatr/rio/filter"
)

const Kind = rio.TransmatKind("file")

var _ rio.Transmat = &Transmat{}

type Transmat struct {
	workPath string
}

var _ rio.TransmatFactory = New

func New(workPath string) rio.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(rio.TransmatError.New("Unable to set up workspace: %s", err))
	}
	return &Transmat{workPath}
}

var hasherFactory = sha512.New384

/*
	Arenas produced by File Transmats may be relocated by simple `mv`.
*/
func (t *Transmat) Materialize(
	kind rio.TransmatKind,
	dataHash rio.CommitID,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.Arena {
	var arena fileArena
	try.Do(func() {
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Before we eval all config, prepend some default filter setup.
		//  We need these defaults here because "keep" isn't a
		//   semantically valid concept (there's no metadata to keep!).
		options = append([]rio.MaterializerConfigurer{
			rio.UseFilter(filter.MtimeFilter{def.FilterDefaultMtime}),
			rio.UseFilter(filter.UidFilter{def.FilterDefaultUid}),
			rio.UseFilter(filter.GidFilter{def.FilterDefaultGid}),
		}, options...)

		// Compile config
		config := rio.EvaluateConfig(options...)

		// Ping silos
		if len(siloURIs) < 1 {
			panic(rio.ConfigError.New("Materialization requires at least one data source!"))
			// Note that it's possible a caching layer will satisfy things even without data sources...
			//  but if that was going to happen, it already would have by now.
		}
		// Our policy is to take the first path that exists.
		//  This lets you specify a series of potential locations,
		//  and if one is unavailable we'll just take the next.
		var warehouse *Warehouse
		for _, uri := range siloURIs {
			wh := NewWarehouse(uri)
			if err := wh.Ping(); err == nil {
				warehouse = wh
				break
			} else {
				log.Info("Warehouse not available, skipping",
					"warehouse", uri,
					"reason", err,
				)
			}
		}
		if warehouse == nil {
			panic(rio.WarehouseUnavailableError.New("No warehouses were available!"))
		}

		// Create staging arena to produce data into.
		f, err := ioutil.TempFile(t.workPath, "")
		if err != nil {
			panic(rio.TransmatError.New("Unable to create arena: %s", err))
		}
		f.Close() // none of the rest of our apis expect the file to already be open, so.
		// Dance filenames.  (fs.PlaceFile uses O_EXCL.  maybe we should patch it with more params.)
		arena.path = f.Name() + ".file"
		defer os.Remove(f.Name())

		// Beginye the fetchery
		stream := warehouse.makeReader(dataHash)
		defer stream.Close()

		// Hash the bare file.  there's no tree, so it's this simple.
		hasher := hasherFactory()
		reader := &flak.HashingReader{stream, hasher}
		fs.PlaceFile(
			arena.Path(),
			config.FilterSet.Apply(fs.Metadata{
				Typeflag:   '0', // tar.TypeReg
				Mode:       0644,
				AccessTime: fs.Epochwhen,
			}),
			reader,
		)
		actualTreeHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		// verify total integrity
		expectedTreeHash := string(dataHash)
		if actualTreeHash == expectedTreeHash {
			// excellent, got what we asked for.
			arena.hash = dataHash
		} else {
			// this may or may not be grounds for panic, depending on configuration.
			if config.AcceptHashMismatch {
				// if we're tolerating mismatches, report the actual hash through different mechanisms.
				// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
				arena.hash = rio.CommitID(actualTreeHash)
			} else {
				panic(rio.NewHashMismatchError(string(dataHash), actualTreeHash))
			}
		}
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return arena
}

func (t Transmat) Scan(
	kind rio.TransmatKind,
	subjectPath string,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.CommitID {
	// NYI because I'm blowing a fuse on "this needs refactor" for the IO components of all the transmats.
	panic(errors.NotImplementedError.New("saving not yet implemented for this transmat"))
}

type fileArena struct {
	path string
	hash rio.CommitID
}

func (a fileArena) Path() string {
	return a.path
}

func (a fileArena) Hash() rio.CommitID {
	return a.hash
}

// rm's.
// does not consider it an error if path already does not exist.
func (a fileArena) Teardown() {
	if err := os.Remove(a.path); err != nil {
		if e2, ok := err.(*os.PathError); ok && e2.Err == syscall.ENOENT && e2.Path == a.path {
			return
		}
		panic(err)
	}
}
