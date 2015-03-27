package tar

import (
	"os"
	"os/exec"

	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
)

const Type = "tar"

// interface assertion
var _ input.Input = &Input{}

type Input struct {
	spec def.Input
}

func New(spec def.Input) *Input {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This input implementation supports definitions of type %q, not %q", Type, spec.Type))
	}
	return &Input{
		spec: spec,
	}
}

func (i Input) Apply(path string) <-chan error {
	done := make(chan error)
	go func() {
		defer close(done)

		// Eventually:
		// iterate along with the tar stream as long as possible
		// if it's out of order, or we hit the same file twice, balk; start again later with the fs
		// (we're not just doing a checksum of the tar as it stands;
		// we want something that's unambiguously reproducable,
		// and tarsum as made by docker isn't that since it just accepts whatever iteration order the tarball has.)

		// Currently:
		// Exec-wrap tar, like a boss

		err := os.MkdirAll(path, 0777)
		if err != nil {
			done <- Error.Wrap(errors.IOError.Wrap(err))
			return
		}

		tar := exec.Command("tar", "-xf", i.spec.URI, "-C", path)
		tar.Stdin = os.Stdin
		tar.Stdout = os.Stdout
		tar.Stderr = os.Stderr

		err = tar.Run()
		if err != nil {
			done <- Error.Wrap(err)
			return
		}

	}()
	return done
}

var Error *errors.ErrorClass = input.Error.NewClass("TarInputError") // currently contains little information because the returns of the subcommand are already opaque.  may become the root of a more expressive error hierarchy when we replace the tar implementation.
