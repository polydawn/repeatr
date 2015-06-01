package tarexec

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	tar_in "polydawn.net/repeatr/input/tar"
	"polydawn.net/repeatr/io"
)

const Kind = integrity.TransmatKind("tar")

var _ integrity.Transmat = &TarExecTransmat{}

type TarExecTransmat struct {
	workPath string
}

var _ integrity.TransmatFactory = New

func New(workPath string) integrity.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(integrity.TransmatError.New("Unable to set up workspace: %s", err))
	}
	return &TarExecTransmat{workPath}
}

/*
	Arenas produced by Dir Transmats may be relocated by simple `mv`.
*/
func (t *TarExecTransmat) Materialize(
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

func (t TarExecTransmat) Scan(
	kind integrity.TransmatKind,
	subjectPath string,
	siloURIs []integrity.SiloURI,
	options ...integrity.MaterializerConfigurer,
) integrity.CommitID {
	try.Do(func() {
		// Basic validation and config
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Parse save locations.
		//  (Most transmats do... significantly smarter things than this backwater.)
		var localPath string
		if len(siloURIs) == 0 {
			localPath = "/dev/null"
		} else if len(siloURIs) == 1 {
			// TODO still assuming all local paths and not doing real uri parsing
			localPath = string(siloURIs[0])
			err := os.MkdirAll(filepath.Dir(localPath), 0755)
			if err != nil {
				panic(integrity.WarehouseConnectionError.New("Unable to write file: %s", err))
			}
			file, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(integrity.WarehouseConnectionError.New("Unable to write file: %s", err))
			}
			file.Close() // just checking, so we can (try to) give a more pleasant error than tar barf
		} else {
			panic(integrity.ConfigError.New("%s transmat only supports shipping to 1 warehouse", Kind))
		}

		// exec tar.
		// in case of a zero (a.k.a. success) exit, this returns silently.
		// in case of a non-zero exit, this panics; the panic will include the output.
		gosh.Gosh(
			"tar",
			"-cf", localPath,
			"--xform", "s,"+strings.TrimLeft(subjectPath, "/")+",.,",
			subjectPath,
			gosh.NullIO,
		).RunAndReport()
	}).Catch(integrity.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(integrity.UnknownError.Wrap(err))
	}).Done()
	return ""

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
