package dir

import (
	"crypto/sha512"
	"hash"
	"os"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/fshash"
)

const Type = "dir"

// interface assertion
var _ input.Input = &Input{}

type Input struct {
	spec          def.Input
	hasherFactory func() hash.Hash
}

func New(spec def.Input) *Input {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This input implementation supports definitions of type %q, not %q", Type, spec.Type))
	}
	_, err := os.Stat(spec.URI)
	if os.IsNotExist(err) {
		panic(def.ValidationError.New("Input URI %q must be a directory", spec.URI))
	}
	return &Input{
		spec:          spec,
		hasherFactory: sha512.New384,
	}
}

func (i Input) Apply(destinationRoot string) <-chan error {
	done := make(chan error)
	go func() {
		defer close(done)

		// walk filesystem, copying and accumulating data for integrity check
		bucket := &fshash.MemoryBucket{}
		err := fshash.FillBucket(i.spec.URI, destinationRoot, bucket, i.hasherFactory)
		if err != nil {
			done <- err
			return
		}

		// hash whole tree and verify total integrity
		// TODO
	}()
	return done
}
