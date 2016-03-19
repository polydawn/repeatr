package dir

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/filter"
	"polydawn.net/repeatr/lib/fshash"
)

const Kind = rio.TransmatKind("dir")

var _ rio.Transmat = &DirTransmat{}

type DirTransmat struct {
	workPath string
}

var _ rio.TransmatFactory = New

func New(workPath string) rio.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(rio.TransmatError.New("Unable to set up workspace: %s", err))
	}
	return &DirTransmat{workPath}
}

var hasherFactory = sha512.New384

/*
	Arenas produced by Dir Transmats may be relocated by simple `mv`.
*/
func (t *DirTransmat) Materialize(
	kind rio.TransmatKind,
	dataHash rio.CommitID,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.Arena {
	var arena dirArena
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
		//  This lets you specify a series of potential locations,
		//  and if one is unavailable we'll just take the next.
		var warehouse *Warehouse
		for _, uri := range siloURIs {
			wh := NewWarehouse(uri)
			if wh.Ping() {
				warehouse = wh
				break
			} else {
				log.Info("Warehouse does not exist, skipping", "warehouse", uri)
			}
		}
		if warehouse == nil {
			panic(rio.WarehouseUnavailableError.New("No warehouses were available!"))
		}

		// Create staging arena to produce data into.
		var err error
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(rio.TransmatError.New("Unable to create arena: %s", err))
		}

		// walk filesystem, copying and accumulating data for integrity check
		hasherFactory := sha512.New384
		bucket := &fshash.MemoryBucket{}
		if err := fshash.FillBucket(warehouse.GetShelf(dataHash), arena.Path(), bucket, filter.FilterSet{}, hasherFactory); err != nil {
			panic(err)
		}

		// hash whole tree
		actualTreeHash := fshash.Hash(bucket, hasherFactory)

		// verify total integrity
		expectedTreeHash, err := base64.URLEncoding.DecodeString(string(dataHash))
		if err != nil {
			panic(rio.ConfigError.New("Could not parse hash: %s", err))
		}
		if bytes.Equal(actualTreeHash, expectedTreeHash) {
			// excellent, got what we asked for.
			arena.hash = dataHash
		} else {
			// this may or may not be grounds for panic, depending on configuration.
			if config.AcceptHashMismatch {
				// if we're tolerating mismatches, report the actual hash through different mechanisms.
				// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
				arena.hash = rio.CommitID(actualTreeHash)
			} else {
				panic(rio.NewHashMismatchError(string(dataHash), base64.URLEncoding.EncodeToString(actualTreeHash)))
			}
		}
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return arena
}

func (t DirTransmat) Scan(
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

		saveFn := func(destPath string) {
			// walk filesystem, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			err = fshash.FillBucket(subjectPath, destPath, bucket, config.FilterSet, hasherFactory)
			if err != nil {
				panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
			}
			// hash whole tree
			actualTreeHash := fshash.Hash(bucket, hasherFactory)
			commitID = rio.CommitID(base64.URLEncoding.EncodeToString(actualTreeHash))
		}

		// First... no save locations is a special case: still need to hash.
		if len(siloURIs) == 0 {
			saveFn("")
			return // for no-save, that's it, we're done
		}

		// Dial warehouses.
		warehouses := make([]*Warehouse, 0, len(siloURIs))
		for _, uri := range siloURIs {
			wh := NewWarehouse(uri)
			if wh.Ping() {
				warehouses = append(warehouses, wh)
			} else {
				log.Info("Unable to contact a warehouse, skipping it", "warehouse", uri)
			}
		}
		// By default we're tolerant of some warehouses being unresponsive
		//  (mirroring is easy and conflict free, after all), but if
		//   ALL of them are down?  That's bad enough news to stop for.
		if len(warehouses) == 0 {
			// Still, finish out determining the hash.
			saveFn("")
			// This is one of those situations where panicking doesn't fit very well...
			//  there's such a thing as partial progress, and we've got it.
			//   Perhaps in the future we should refactor scan results to include errors
			//    values... per stage, since that gets several birds with one stone.
			panic(rio.WarehouseUnavailableError.New("NO warehouses available -- data not saved!"))
		}

		// Open writers to save locations, and commit to each one.
		//  (We do this serially for now; it could be parallelized, but
		//   the dircopy code wasn't written with multiwriters in mind.)
		for _, warehouse := range warehouses {
			wc := warehouse.openWriter()
			saveFn(wc.tmpPath)
			wc.commit(commitID)
			log.Info("Commited to warehouse", "warehouse", warehouse.localPath, "hash", commitID)
		}
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return commitID
}

type dirArena struct {
	path string
	hash rio.CommitID
}

func (a dirArena) Path() string {
	return a.path
}

func (a dirArena) Hash() rio.CommitID {
	return a.hash
}

// rm's.
// does not consider it an error if path already does not exist.
func (a dirArena) Teardown() {
	if err := os.RemoveAll(a.path); err != nil {
		if e2, ok := err.(*os.PathError); ok && e2.Err == syscall.ENOENT && e2.Path == a.path {
			return
		}
		panic(err)
	}
}
