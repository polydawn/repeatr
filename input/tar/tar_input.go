package tar

import (
	"os"

	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
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
		try.Do(func() {
			err := os.MkdirAll(path, 0777)
			if err != nil {
				panic(input.TargetFilesystemUnavailableIOError(err))
			}

			// exec tar.
			// in case of a zero (a.k.a. success) exit, this returns silently.
			// in case of a non-zero exit, this panics; the panic will include the output.
			gosh.Gosh(
				"tar",
				"-xf", i.spec.URI,
				"-C", path,
				gosh.NullIO,
			).RunAndReport()

			// note: indeed, we never check the hash field.  this is *not* a compliant implementation of an input.
		}).Catch(input.Error, func(err *errors.Error) {
			done <- err
		}).CatchAll(func(err error) {
			// All errors we emit will be under `input.Error`'s type.
			done <- input.UnknownError.Wrap(err)
		}).Done()
	}()
	return done
}
