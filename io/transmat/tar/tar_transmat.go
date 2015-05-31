package tar

import (
	"crypto/sha512"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	tar_in "polydawn.net/repeatr/input/tar2"
	"polydawn.net/repeatr/io"
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
	config := integrity.EvaluateConfig(options...)
	var err error
	var arena tarArena
	arena.path, err = ioutil.TempDir(t.workPath, "")
	if err != nil {
		panic(integrity.TransmatError.New("Unable to create arena: %s", err))
	}
	arena.hash = dataHash // until proven otherwise
	err = <-tar_in.New(def.Input{
		// Wrapping around previous implementations until we migrate it all.
		// Ugly, but will let us migrate consumer apis next, then unwind these wrappers,
		// and thus never break things while in flight.
		Type: string(kind),
		Hash: string(dataHash),
		URI:  string(siloURIs[0]),
	}).Apply(arena.path)
	// Also ugly!  When we unwind these wrappers, everything will
	// consistently be blocking behaviors, and this will clean up substantially.
	if err != nil {
		if config.AcceptHashMismatch && errors.GetClass(err).Is(integrity.HashMismatchError) {
			// if we're tolerating mismatches, report the actual hash through different mechanisms.
			// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
			arena.hash = integrity.CommitID(errors.GetData(err, integrity.HashActualKey).(string))
		} else {
			panic(err)
		}
	}
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
