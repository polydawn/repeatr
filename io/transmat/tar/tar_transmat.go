package tar

import (
	"archive/tar"
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
)

const Kind = integrity.TransmatKind("tar")

var _ integrity.Transmat = &TarTransmat{}

type TarTransmat struct {
	workPath string
}

var _ integrity.TransmatFactory = New

func New(workPath string) integrity.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(integrity.TransmatError.New("Unable to set up workspace: %s", err))
	}
	return &TarTransmat{workPath}
}

var hasherFactory = sha512.New384

/*
	Arenas produced by Tar Transmats may be relocated by simple `mv`.
*/
func (t *TarTransmat) Materialize(
	kind integrity.TransmatKind,
	dataHash integrity.CommitID,
	siloURIs []integrity.SiloURI,
	options ...integrity.MaterializerConfigurer,
) integrity.Arena {
	var arena tarArena
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
		//  This lets you specify a series of potential locations, and if one is unavailable we'll just take the next.
		var siloURI integrity.SiloURI
		for _, givenURI := range siloURIs {
			// TODO still assuming all local paths and not doing real uri parsing
			localPath := string(givenURI)
			_, err := os.Stat(localPath)
			if os.IsNotExist(err) {
				// TODO it'd be awfully lovely if we could log the attempt somewhere
				continue
			}
			siloURI = givenURI
			break
		}
		if siloURI == "" {
			panic(integrity.WarehouseConnectionError.New("No warehouses were available!"))
		}
		// Open the input stream; preparing decompression as necessary
		file, err := os.OpenFile(string(siloURI), os.O_RDONLY, 0755)
		if err != nil {
			panic(integrity.WarehouseConnectionError.New("Unable to read file: %s", err))
		}
		defer file.Close()
		reader, err := Decompress(file)
		if err != nil {
			panic(integrity.WarehouseConnectionError.New("could not start decompressing: %s", err))
		}
		tarReader := tar.NewReader(reader)

		// Create staging arena to produce data into.
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(integrity.TransmatError.New("Unable to create arena: %s", err))
		}

		// walk input tar stream, placing data and accumulating hashes and metadata for integrity check
		bucket := &fshash.MemoryBucket{}
		Extract(tarReader, arena.Path(), bucket, hasherFactory)

		// bucket processing may have created a root node if missing.  if so, we need to apply its props.
		fs.PlaceFile(arena.Path(), bucket.Root().Metadata, nil)

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

func (t TarTransmat) Scan(
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

		// Open output streams for writing.
		// Since these are all behaving as just one `io.Writer` stream, this could maybe be factored out.
		// Error handling is currently "anything -> panic".  This should probably be more resilient.  (That might need another refactor so we have an upload call per remote.)
		writers := make([]io.Writer, 0)
		closers := make([]io.Closer, 0)
		for _, givenURI := range siloURIs {
			// TODO still assuming all local paths and not doing real uri parsing
			file, err := os.OpenFile(string(givenURI), os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(integrity.WarehouseConnectionError.New("Unable to write file: %s", err))
			}
			writers = append(writers, file)
			closers = append(closers, file)
		}
		defer func() {
			for _, closer := range closers {
				if err := closer.Close(); err != nil {
					panic(integrity.WarehouseConnectionError.New("Unable to close file: %s", err))
				}
			}
		}()
		stream := io.MultiWriter(writers...)
		if len(writers) < 1 {
			stream = ioutil.Discard
		}

		// walk, fwrite, hash
		commitID = integrity.CommitID(Save(stream, subjectPath, hasherFactory))
	}).Catch(integrity.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(integrity.UnknownError.Wrap(err))
	}).Done()
	return commitID
}

type tarArena struct {
	path string
	hash integrity.CommitID
}

func (a tarArena) Path() string {
	return a.path
}

func (a tarArena) Hash() integrity.CommitID {
	return a.hash
}

// rm's.
// does not consider it an error if path already does not exist.
func (a tarArena) Teardown() {
	if err := os.RemoveAll(a.path); err != nil {
		if e2, ok := err.(*os.PathError); ok && e2.Err == syscall.ENOENT && e2.Path == a.path {
			return
		}
		panic(err)
	}
}
