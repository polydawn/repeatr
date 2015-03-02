package tar

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
)

const Type = "tar"

// interface assertion
var _ input.Input = &Input{}

type Input struct {
}

func New(spec def.Input) *Input {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This input implementation supports definitions of type %q, not %q", Type, spec.Type))
	}
	return &Input{}
}

func (Input) Apply(path string) <-chan error {
	done := make(chan error)
	go func() {
		defer close(done)
		// iterate along with the tar stream as long as possible
		// if it's out of order, or we hit the same file twice, balk; start again later with the fs
		// (we're not just doing a checksum of the tar as it stands;
		// we want something that's unambiguously reproducable,
		// and tarsum as made by docker isn't that since it just accepts whatever iteration order the tarball has.)
	}()
	return done
}
