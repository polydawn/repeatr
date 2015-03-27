package tar2

import (
	"crypto/sha512"
	"encoding/base64"
	"hash"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/output"
)

const Type = "tar"

var _ output.Output = &Output{} // interface assertion

type Output struct {
	spec          def.Output
	hasherFactory func() hash.Hash
}

func New(spec def.Output) *Output {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This output implementation supports definitions of type %q, not %q", Type, spec.Type))
	}
	return &Output{
		spec:          spec,
		hasherFactory: sha512.New384,
	}
}

func (o Output) Apply(basePath string) <-chan error {
	done := make(chan error)
	go func() {
		defer close(done)
		try.Do(func() {
			// walk filesystem, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			// TODO STUFF

			// hash whole tree
			actualTreeHash, _ := fshash.Hash(bucket, o.hasherFactory)

			// report the hash by mutating our spec object.
			// heh, see the problem there?  pointers
			o.spec.Hash = base64.URLEncoding.EncodeToString(actualTreeHash)
		}).CatchAll(func(e error) {
			// any errors should be caught and forwarded through a channel for management rather than killing the whole sytem unexpectedly.
			// this could maybe be improved with better error grouping (wrap all errors in a type that indicates origin subsystem and forwards the original as a 'cause' attachement, etc).
			done <- e
		}).Done()
	}()
	return done
}
