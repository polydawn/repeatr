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
	var arena dirArena
	try.Do(func() {
		// Basic validation and config
		if !(kind == Kind || kind == "exec-tar") {
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
		file.Close() // just checking, so we can (try to) give a more pleasant error than tar barf

		// Create staging arena to produce data into.
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(integrity.TransmatError.New("Unable to create arena: %s", err))
		}

		// exec tar.
		// in case of a zero (a.k.a. success) exit, this returns silently.
		// in case of a non-zero exit, this panics; the panic will include the output.
		gosh.Gosh(
			"tar",
			"-xf", string(siloURI),
			"-C", arena.Path(),
			gosh.NullIO,
		).RunAndReport()

		// note: indeed, we never check the hash field.  this is *not* a compliant implementation of an input.
	}).Catch(integrity.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(integrity.UnknownError.Wrap(err))
	}).Done()
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
		panic(integrity.TransmatError.New("Failed to tear down arena: %s", err))
	}
}
