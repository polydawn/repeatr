package s3

import (
	"crypto/sha512"
	"hash"
	"io"
	"time"

	"github.com/rlmcpherson/s3gof3r"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/output/tar2"
)

const Type = "s3"

var _ output.Output = &Output{} // interface assertion

/*
	Amazon S3 silos are used by this transport in a very MVP-oriented way:
	tarballs streamed in and out, because we know how to preserve file
	attributes in a widely-understood way by doing this.

	This IO system happens to share the same hash-space as the Tar IO system,
	and may thus safely share a cache with Tar IO systems.
*/
type Output struct {
	spec          def.Output
	hasherFactory func() hash.Hash
}

func New(spec def.Output) output.Output {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This output implementation supports definitions of type %q, not %q", Type, spec.Type))
	}
	return &Output{
		spec:          spec,
		hasherFactory: sha512.New384,
	}
}

func (o Output) Apply(basePath string) <-chan output.Report {
	// We actually shell out to the entire streaming part of the tar system.
	// All the formatting and hashing is identical; this just shoves the
	//  stream to a S3 bucket instead of a local filesystem.
	done := make(chan output.Report)
	go func() {
		defer close(done)
		try.Do(func() {
			// parse URI
			// TODO
			bucketName := "repeatr-test"
			storePath := "keks"

			// load keys from env
			// TODO someday URIs should grow smart enough to control this in a more general fashion -- but for now, host ENV is actually pretty feasible and plays easily with others.
			keys, err := s3gof3r.EnvKeys()
			if err != nil {
				panic(S3ConfigurationMissingError.Wrap(err))
			}

			// initialize writer to s3!
			s3writer := makeS3writer(bucketName, storePath, keys)
			defer s3writer.Close()

			// walk, fwrite, hash
			o.spec.Hash = tar2.Save(s3writer, basePath, o.hasherFactory)

			// TODO
			// if the URI indicated splay behavior, do a rename from the upload location to final CA resting place.

			done <- output.Report{nil, o.spec}
		}).Catch(output.Error, func(err *errors.Error) {
			done <- output.Report{err, o.spec}
		}).CatchAll(func(err error) {
			// All errors we emit will be under `output.Error`'s type.
			done <- output.Report{output.UnknownError.Wrap(err).(*errors.Error), o.spec}
		}).Done()
	}()
	return done
}

func makeS3writer(bucketName string, path string, keys s3gof3r.Keys) io.WriteCloser {
	conf := &s3gof3r.Config{
		Concurrency: 10,
		PartSize:    20 * 1024 * 1024,
		NTry:        10,
		Md5Check:    false,
		Scheme:      "https",
		Client:      s3gof3r.ClientWithTimeout(5 * time.Second),
	}
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	bucket := s3.Bucket(bucketName)
	w, err := bucket.PutWriter(path, nil, conf)
	if err != nil {
		panic(S3Error.Wrap(err))
	}
	return w
}
