package dir

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fshash"
	dir_out "polydawn.net/repeatr/output/dir"
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
		panic(input.TargetFilesystemUnavailableIOError(err)) // TODO these errors should migrate
	}
	return &DirTransmat{workPath}
}

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
			panic(errors.ProgrammerError.New("This input implementation supports definitions of type %q, not %q", Kind, kind))
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

		// Create staging arena to produce data into.
		var err error
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(input.TargetFilesystemUnavailableIOError(err))
		}

		// walk filesystem, copying and accumulating data for integrity check
		hasherFactory := sha512.New384
		bucket := &fshash.MemoryBucket{}
		localPath := string(siloURI)
		if err := fshash.FillBucket(localPath, arena.Path(), bucket, hasherFactory); err != nil {
			panic(err)
		}

		// hash whole tree
		actualTreeHash, err := fshash.Hash(bucket, hasherFactory)
		if err != nil {
			panic(err)
		}

		// verify total integrity
		expectedTreeHash, err := base64.URLEncoding.DecodeString(string(dataHash))
		if bytes.Equal(actualTreeHash, expectedTreeHash) {
			// excellent, got what we asked for.
			arena.hash = dataHash
		} else {
			// this may or may not be grounds for panic, depending on configuration.
			if config.AcceptHashMismatch && errors.GetClass(err).Is(input.InputHashMismatchError) {
				// if we're tolerating mismatches, report the actual hash through different mechanisms.
				// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
				arena.hash = integrity.CommitID(actualTreeHash)
			} else {
				panic(input.NewHashMismatchError(string(dataHash), base64.URLEncoding.EncodeToString(actualTreeHash)))
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
	if len(siloURIs) <= 0 {
		// odd hack, replace with actual comprehensive of uri lists when finishing migrating.
		// empty strings here make it all the way to the fshash walker, which sees that as a "don't copy" instruction.
		siloURIs = []integrity.SiloURI{""}
	}
	report := <-dir_out.New(def.Output{
		Type: string(kind),
		URI:  string(siloURIs[0]),
	}).Apply(subjectPath)
	if report.Err != nil {
		panic(report.Err)
	}
	return integrity.CommitID(report.Output.Hash)
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
