package tar

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/lib/fshash"
	"go.polydawn.net/repeatr/rio"
)

const Kind = rio.TransmatKind("tar")

var _ rio.Transmat = &TarTransmat{}

type TarTransmat struct {
	workPath string
}

var _ rio.TransmatFactory = New

func New(workPath string) rio.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(rio.TransmatError.New("Unable to set up workspace: %s", err))
	}
	return &TarTransmat{workPath}
}

var hasherFactory = sha512.New384

/*
	Arenas produced by Tar Transmats may be relocated by simple `mv`.
*/
func (t *TarTransmat) Materialize(
	kind rio.TransmatKind,
	dataHash rio.CommitID,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.Arena {
	var arena tarArena
	try.Do(func() {
		// Basic validation and config
		config := rio.EvaluateConfig(options...)
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Ping silos
		if len(siloURIs) < 1 {
			panic(rio.ConfigError.New("Materialization requires at least one data source!"))
			// Note that it's possible a caching layer will satisfy things even without data sources...
			//  but if that was going to happen, it already would have by now.
		}
		// Our policy is to take the first path that exists.
		//  This lets you specify a series of potential locations, and if one is unavailable we'll just take the next.
		var stream io.Reader
		var available bool
		for _, uri := range siloURIs {
			try.Do(func() {
				stream = makeReader(dataHash, uri)
			}).Catch(rio.WarehouseUnavailableError, func(err *errors.Error) {
				// fine, we'll just try the next one
				log.Info("Warehouse does not exist, skipping", "warehouse", uri)
			}).Catch(rio.DataDNE, func(err *errors.Error) {
				// fine, we'll just try the next one
				available = true // but at least someone was *alive*
				log.Info("Warehouse does not have the data, skipping", "warehouse", uri, "hash", dataHash)
			}).Done()
			if stream != nil {
				break
			}
		}
		if stream == nil {
			if available {
				panic(rio.DataDNE.New("No warehouses had the data!"))
			}
			panic(rio.WarehouseUnavailableError.New("No warehouses were available!"))
		}

		// Wrap input stream with decompression as necessary
		reader, err := Decompress(stream)
		if err != nil {
			panic(rio.WarehouseIOError.New("could not start decompressing: %s", err))
		}
		tarReader := tar.NewReader(reader)

		// Create staging arena to produce data into.
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(rio.TransmatError.New("Unable to create arena: %s", err))
		}

		// walk input tar stream, placing data and accumulating hashes and metadata for integrity check
		bucket := &fshash.MemoryBucket{}
		Extract(tarReader, arena.Path(), bucket, hasherFactory, log)

		// bucket processing may have created a root node if missing.  if so, we need to apply its props.
		fs.PlaceFile(arena.Path(), bucket.Root().Metadata, nil)

		// hash whole tree
		actualTreeHash := base64.URLEncoding.EncodeToString(fshash.Hash(bucket, hasherFactory))

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

func (t TarTransmat) Scan(
	kind rio.TransmatKind,
	subjectPath string,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.CommitID {
	var commitID rio.CommitID
	try.Do(func() {
		// Basic validation and config
		config := rio.EvaluateConfig(options...)
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// If scan area doesn't exist, bail immediately.
		// No need to even start dialing warehouses if we've got nothing for em.
		_, err := os.Stat(subjectPath)
		if err != nil {
			if os.IsNotExist(err) {
				return // empty commitID
			} else {
				panic(err)
			}
		}

		// Open output streams for writing.
		// Since these are all behaving as just one `io.Writer` stream, this could maybe be factored out.
		// Error handling is currently "anything -> panic".  This should probably be more resilient.  (That might need another refactor so we have an upload call per remote.)
		controllers := make([]StreamingWarehouseWriteController, 0)
		writers := make([]io.Writer, 0)
		for _, uri := range siloURIs {
			controller := makeWriteController(uri)
			controllers = append(controllers, controller)
			writers = append(writers, controller.Writer())
		}
		stream := io.MultiWriter(writers...)
		if len(writers) < 1 {
			stream = ioutil.Discard
		}

		// walk, fwrite, hash
		commitID = rio.CommitID(Save(stream, subjectPath, config.FilterSet, hasherFactory))

		// commit
		for _, controller := range controllers {
			controller.Commit(commitID)
		}
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return commitID
}

type tarArena struct {
	path string
	hash rio.CommitID
}

func (a tarArena) Path() string {
	return a.path
}

func (a tarArena) Hash() rio.CommitID {
	return a.hash
}

// rm's.
// does not consider it an error if path already does not exist.
func (a tarArena) Teardown() {
	if err := os.RemoveAll(a.path); err != nil {
		if e2, ok := err.(*os.PathError); ok && e2.Err == syscall.ENOENT && e2.Path == a.path {
			return
		}
		panic(rio.TransmatError.New("Failed to tear down arena: %s", err))
	}
}
