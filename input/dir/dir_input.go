package dir

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"os"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/lib/treewalk"
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

		// hash whole tree
		hasher := i.hasherFactory()
		preVisit := func(node treewalk.Node) error {
			record := node.(fshash.RecordIterator).Record()
			metabin, err := record.Metadata.MarshalBinary()
			if err != nil {
				return err
			}
			// TODO : this also needs some higher level alignment/length stuff
			_, err = hasher.Write(metabin)
			if err != nil {
				return err
			}
			_, err = hasher.Write(record.ContentHash)
			return err
		}
		postVisit := func(node treewalk.Node) error { return nil }
		err = treewalk.Walk(bucket.Iterator(), preVisit, postVisit)

		// verify total integrity
		actualTreeHash := hasher.Sum(nil)
		expectedTreeHash, err := base64.URLEncoding.DecodeString(i.spec.Hash)
		if !bytes.Equal(actualTreeHash, expectedTreeHash) {
			done <- input.InputHashMismatchError.New("expected hash %q, got %q", i.spec.Hash, base64.URLEncoding.EncodeToString(actualTreeHash))
		}
	}()
	return done
}
