package tar

import (
	"io/ioutil"
	"os"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	tar_in "polydawn.net/repeatr/input/tar2"
	"polydawn.net/repeatr/io"
	tar_out "polydawn.net/repeatr/output/tar2"
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
		panic(input.TargetFilesystemUnavailableIOError(err)) // TODO these errors should migrate
	}
	return &TarTransmat{workPath}
}

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
		panic(input.TargetFilesystemUnavailableIOError(err))
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

func (t TarTransmat) Scan(
	kind integrity.TransmatKind,
	subjectPath string,
	siloURIs []integrity.SiloURI,
	options ...integrity.MaterializerConfigurer,
) integrity.CommitID {
	if len(siloURIs) <= 0 {
		// odd hack, replace with actual comprehensive of uri lists when finishing migrating.
		siloURIs = []integrity.SiloURI{"/dev/null"}
	}
	report := <-tar_out.New(def.Output{
		Type: string(kind),
		URI:  string(siloURIs[0]),
	}).Apply(subjectPath)
	if report.Err != nil {
		panic(report.Err)
	}
	return integrity.CommitID(report.Output.Hash)
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
