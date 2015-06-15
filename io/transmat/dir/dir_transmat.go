package dir

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fshash"
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
		var siloURI integrity.SiloURI
		for _, givenURI := range siloURIs {
			// TODO still assuming all local paths and not doing real uri parsing
			localPath := string(givenURI)
			_, err := os.Stat(localPath)
			if os.IsNotExist(err) {
				continue
			}
			siloURI = givenURI
			break
		}
		if siloURI == "" {
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
		localPath := string(siloURI)
		if err := fshash.FillBucket(localPath, arena.Path(), bucket, hasherFactory); err != nil {
			panic(err)
		}

		// hash whole tree
		actualTreeHash := fshash.Hash(bucket, hasherFactory)

		// verify total integrity
		expectedTreeHash, err := base64.URLEncoding.DecodeString(string(dataHash))
		if bytes.Equal(actualTreeHash, expectedTreeHash) {
			// excellent, got what we asked for.
			arena.hash = dataHash
		} else {
			// this may or may not be grounds for panic, depending on configuration.
			if config.AcceptHashMismatch && errors.GetClass(err).Is(integrity.HashMismatchError) {
				// if we're tolerating mismatches, report the actual hash through different mechanisms.
				// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
				arena.hash = integrity.CommitID(actualTreeHash)
			} else {
				panic(err)
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
	options ...integrity.MaterializerConfigurer,
) integrity.CommitID {
	var commitID integrity.CommitID
	try.Do(func() {
		// Basic validation and config
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Parse save locations.
		// This transmat only supports one output location at a time due
		//  to Old code we haven't invested in refactoring yet.
		var localPath string
		if len(siloURIs) == 0 {
			localPath = "" // empty string is a well known value to `fshash.FillBucket`: means just hash, don't copy.
		} else if len(siloURIs) == 1 {
			// TODO still assuming all local paths and not doing real uri parsing
			localPath = string(siloURIs[0])
			err := os.MkdirAll(filepath.Dir(localPath), 0755)
			if err != nil {
				panic(integrity.WarehouseConnectionError.New("Unable to write file: %s", err))
			}
		} else {
			panic(integrity.ConfigError.New("%s transmat only supports shipping to 1 warehouse", Kind))
		}

		// walk filesystem, copying and accumulating data for integrity check
		bucket := &fshash.MemoryBucket{}
		err := fshash.FillBucket(subjectPath, localPath, bucket, hasherFactory)
		if err != nil {
			panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
		}

		// hash whole tree
		actualTreeHash := fshash.Hash(bucket, hasherFactory)

		// report
		commitID = integrity.CommitID(base64.URLEncoding.EncodeToString(actualTreeHash))
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
