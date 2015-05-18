package s3

import (
	"io/ioutil"
	"os"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	s3_in "polydawn.net/repeatr/input/s3"
	"polydawn.net/repeatr/io"
	s3_out "polydawn.net/repeatr/output/s3"
)

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
	config := integrity.EvaluateConfig(options...)
	var err error
	var arena dirArena
	arena.path, err = ioutil.TempDir(t.workPath, "")
	if err != nil {
		panic(input.TargetFilesystemUnavailableIOError(err))
	}
	arena.hash = dataHash // until proven otherwise
	err = <-s3_in.New(def.Input{
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
		if config.AcceptHashMismatch && errors.GetClass(err).Is(input.InputHashMismatchError) {
			// if we're tolerating mismatches, report the actual hash through different mechanisms.
			// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
			arena.hash = integrity.CommitID(errors.GetData(err, input.HashActualKey).(string))
		} else {
			panic(err)
		}
	}
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
		siloURIs = []integrity.SiloURI{"null://"}
	}
	report := <-s3_out.New(def.Output{
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
