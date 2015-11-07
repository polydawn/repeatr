package dir

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/filter"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/lib/guid"
)

const Kind = integrity.TransmatKind("dir")

var _ integrity.Transmat = &DirTransmat{}

type DirTransmat struct {
	workPath string
}

var _ integrity.TransmatFactory = New

func New(workPath string) integrity.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(integrity.TransmatError.New("Unable to set up workspace: %s", err))
	}
	return &DirTransmat{workPath}
}

var hasherFactory = sha512.New384

/*
	Arenas produced by Dir Transmats may be relocated by simple `mv`.
*/
func (t *DirTransmat) Materialize(
	kind integrity.TransmatKind,
	dataHash integrity.CommitID,
	siloURIs []integrity.SiloURI,
	log log15.Logger,
	options ...integrity.MaterializerConfigurer,
) integrity.Arena {
	var arena dirArena
	try.Do(func() {
		// Basic validation and config
		config := integrity.EvaluateConfig(options...)
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Ping silos
		if len(siloURIs) < 1 {
			panic(integrity.ConfigError.New("Materialization requires at least one data source!"))
			// Note that it's possible a caching layer will satisfy things even without data sources...
			//  but if that was going to happen, it already would have by now.
		}
		// Our policy is to take the first path that exists.
		//  This lets you specify a series of potential locations,
		var localSourcePath string
		for _, givenURI := range siloURIs {
			pth := reckonCommittedPath(dataHash, givenURI)
			_, err := os.Stat(pth)
			if os.IsNotExist(err) {
				log.Info("Warehouse does not exist, skipping", "warehouse", givenURI)
				continue
			}
			localSourcePath = pth
			break
		}
		if localSourcePath == "" {
			panic(integrity.WarehouseConnectionError.New("No warehouses were available!"))
		}

		// Create staging arena to produce data into.
		var err error
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(integrity.TransmatError.New("Unable to create arena: %s", err))
		}

		// walk filesystem, copying and accumulating data for integrity check
		hasherFactory := sha512.New384
		bucket := &fshash.MemoryBucket{}
		if err := fshash.FillBucket(localSourcePath, arena.Path(), bucket, filter.FilterSet{}, hasherFactory); err != nil {
			panic(err)
		}

		// hash whole tree
		actualTreeHash := fshash.Hash(bucket, hasherFactory)

		// verify total integrity
		expectedTreeHash, err := base64.URLEncoding.DecodeString(string(dataHash))
		if err != nil {
			panic(integrity.ConfigError.New("Could not parse hash: %s", err))
		}
		if bytes.Equal(actualTreeHash, expectedTreeHash) {
			// excellent, got what we asked for.
			arena.hash = dataHash
		} else {
			// this may or may not be grounds for panic, depending on configuration.
			if config.AcceptHashMismatch {
				// if we're tolerating mismatches, report the actual hash through different mechanisms.
				// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
				arena.hash = integrity.CommitID(actualTreeHash)
			} else {
				panic(integrity.NewHashMismatchError(string(dataHash), base64.URLEncoding.EncodeToString(actualTreeHash)))
			}
		}
	}).Catch(integrity.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(integrity.UnknownError.Wrap(err))
	}).Done()
	return arena
}

func (t DirTransmat) Scan(
	kind integrity.TransmatKind,
	subjectPath string,
	siloURIs []integrity.SiloURI,
	log log15.Logger,
	options ...integrity.MaterializerConfigurer,
) integrity.CommitID {
	var commitID integrity.CommitID
	try.Do(func() {
		// Basic validation and config
		config := integrity.EvaluateConfig(options...)
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

		// First... no save locations is a special case: still need to hash.
		var actualTreeHash []byte
		if len(siloURIs) == 0 {
			// walk filesystem, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			err = fshash.FillBucket(subjectPath, "", bucket, config.FilterSet, hasherFactory)
			if err != nil {
				panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
			}
			// hash whole tree
			actualTreeHash = fshash.Hash(bucket, hasherFactory)
			commitID = integrity.CommitID(base64.URLEncoding.EncodeToString(actualTreeHash))
			// for no-save, that's it, we're done
			return
		}

		// Parse save locations.
		for _, givenURI := range siloURIs {
			stagedDestPath := claimPrecommitPath(givenURI)

			// walk filesystem, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			err = fshash.FillBucket(subjectPath, stagedDestPath, bucket, config.FilterSet, hasherFactory)
			if err != nil {
				panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
			}
			// hash whole tree
			actualTreeHash = fshash.Hash(bucket, hasherFactory)
			commitID = integrity.CommitID(base64.URLEncoding.EncodeToString(actualTreeHash))

			// commit into place
			destPath := reckonCommittedPath(commitID, givenURI)
			err := os.Rename(stagedDestPath, destPath)
			if err != nil {
				// TODO this should probably accept races and just move on, in CA mode anyway.
				// In non-CA mode, this should only happen in case of misconfig or
				// racey use (in which case as usual, you're already Doing It Wrong and we're just being frank about it).
				panic(integrity.WarehouseConnectionError.New("failed moving data to committed location: %s", err))
			}
			// if not using a CA mode, we destroy the previous data... this is no different than, say, tar's behavior, but also a little scary.  use CA mode, ffs.
			// TODO cleanup
		}
	}).Catch(integrity.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(integrity.UnknownError.Wrap(err))
	}).Done()
	return commitID
}

type dirArena struct {
	path string
	hash integrity.CommitID
}

func (a dirArena) Path() string {
	return a.path
}

func (a dirArena) Hash() integrity.CommitID {
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

/*
	Returns a local file path as a string (dir transmat doesn't work any other way).
*/
func reckonCommittedPath(dataHash integrity.CommitID, warehouseCoords integrity.SiloURI) string {
	u, err := url.Parse(string(warehouseCoords))
	if err != nil {
		panic(integrity.ConfigError.New("failed to parse URI: %s", err))
	}
	switch u.Scheme {
	case "file+ca":
		u.Path = filepath.Join(u.Path, string(dataHash))
		fallthrough
	case "file":
		return filepath.Join(u.Host, u.Path) // file uris don't have hosts
	case "":
		panic(integrity.ConfigError.New("missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\""))
	default:
		panic(integrity.ConfigError.New("unsupported scheme in warehouse URI: %q", u.Scheme))
	}
}

/*
	Returns a local file path to a tempdir for writing pre-commit data to.

	May raise WarehouseConnectionError if it can't write; the tempdir is
	created using the appropriate atomic mechanisms (e.g. we do touch
	the disk before returning the path string).
*/
func claimPrecommitPath(warehouseCoords integrity.SiloURI) string {
	u, err := url.Parse(string(warehouseCoords))
	if err != nil {
		panic(integrity.ConfigError.New("failed to parse URI: %s", err))
	}
	var precommitPath string
	switch u.Scheme {
	case "file+ca":
		pathPrefix := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		precommitPath, err = ioutil.TempDir(pathPrefix, ".tmp.upload."+guid.New()+".")
	case "file":
		pathPrefix := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		precommitPath, err = ioutil.TempDir(filepath.Dir(pathPrefix), ".tmp.upload."+filepath.Base(pathPrefix)+"."+guid.New()+".")
	case "":
		panic(integrity.ConfigError.New("missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\""))
	default:
		panic(integrity.ConfigError.New("unsupported scheme in warehouse URI: %q", u.Scheme))
	}
	if err != nil {
		panic(integrity.WarehouseConnectionError.New("failed to reserve temp space in warehouse: %s", err))
	}
	return precommitPath
}
