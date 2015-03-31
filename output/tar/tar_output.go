package tar

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spacemonkeygo/errors"

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

func (i Output) Apply(rootPath string) <-chan output.Report {
	done := make(chan output.Report)
	go func() {
		defer close(done)

		err := os.MkdirAll(rootPath, 0777)
		if err != nil {
			done <- output.Report{errors.IOError.Wrap(err).(*errors.Error), def.Output{}}
		}

		// Assumes output URI is a folder. Output transport impls should obviously be more robust
		path := filepath.Join(rootPath, i.spec.Location)
		tar := exec.Command("tar", "-cf", i.spec.URI, "--xform", "s,"+strings.TrimLeft(rootPath, "/")+",,", path)

		//  path
		tar.Stdin = os.Stdin
		tar.Stdout = os.Stdout
		tar.Stderr = os.Stderr

		err = tar.Run()
		if err != nil {
			done <- output.Report{output.UnknownError.Wrap(err).(*errors.Error), def.Output{}}
			return
		}
	}()
	return done
}
