package tar

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output"
)

const Type = "tar"

// interface assertion
var _ output.Output = &Output{}

type Output struct {
	spec def.Output
}

func New(spec def.Output) *Output {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This output implementation supports definitions of type %q, not %q", Type, spec.Type))
	}
	return &Output{
		spec: spec,
	}
}

func (o Output) Apply(rootPath string) <-chan output.Report {
	done := make(chan output.Report)
	go func() {
		defer close(done)
		try.Do(func() {
			err := os.MkdirAll(rootPath, 0777)
			if err != nil {
				panic(output.TargetFilesystemUnavailableIOError(err))
			}

			// Assumes output URI is a folder. Output transport impls should obviously be more robust
			path := filepath.Join(rootPath, o.spec.Location)

			// exec tar.
			// in case of a zero (a.k.a. success) exit, this returns silently.
			// in case of a non-zero exit, this panics; the panic will include the output.
			gosh.Gosh(
				"tar",
				"-cf", o.spec.URI,
				"--xform", "s,"+strings.TrimLeft(rootPath, "/")+",,",
				path,
				gosh.NullIO,
			).RunAndReport()

			// report
			// note: indeed, we never set the hash field.  this is *not* a compliant implementation of an output.
			done <- output.Report{nil, o.spec}
		}).Catch(output.Error, func(err *errors.Error) {
			done <- output.Report{err, o.spec}
		}).CatchAll(func(err error) {
			// All errors we emit will be under `output.Error`'s type.
			// Every time we hit this UnknownError path, we should consider it a bug until that error is categorized.
			done <- output.Report{output.UnknownError.Wrap(err).(*errors.Error), o.spec}
		}).Done()
	}()
	return done
}
