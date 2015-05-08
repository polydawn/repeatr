package dir

import (
	"io/ioutil"
	"os"
	"syscall"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	dir_in "polydawn.net/repeatr/input/dir"
	"polydawn.net/repeatr/io"
	dir_out "polydawn.net/repeatr/output/dir"
)

var _ integrity.Transmat = &DirTransmat{}

type DirTransmat struct {
	workPath string
}

var _ integrity.TransmatFactory = New

func New(workPath string) integrity.Transmat {
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
	var err error
	var arena dirArena
	arena.path, err = ioutil.TempDir(t.workPath, "")
	if err != nil {
		panic(input.TargetFilesystemUnavailableIOError(err))
	}
	err = <-dir_in.New(def.Input{
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
		panic(err)
	}
	return arena
}

func (t DirTransmat) Scan(kind integrity.TransmatKind, subjectPath string, siloURIs []integrity.SiloURI, options ...integrity.MaterializerConfigurer) integrity.CommitID {
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
}

func (a dirArena) Path() string {
	return a.path
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
