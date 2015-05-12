package s3

import (
	"archive/tar"
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"io"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/rlmcpherson/s3gof3r"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/input/tar2"
	"polydawn.net/repeatr/lib/fshash"
)

const Type = "s3"

var _ input.Input = &Input{} // interface assertion

type Input struct {
	spec          def.Input
	hasherFactory func() hash.Hash
}

func New(spec def.Input) input.Input {
	if spec.Type != Type {
		panic(errors.ProgrammerError.New("This input implementation supports definitions of type %q, not %q", Type, spec.Type))
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
		try.Do(func() {
			// do make a dir for untaring into.
			// tars may specify permission and time bits for their top dir, but if not, we'll start with sane defaults.
			err := os.MkdirAll(destinationRoot, 0755)
			if err != nil {
				panic(input.TargetFilesystemUnavailableIOError(err))
			}

			// parse URI
			u, err := url.Parse(i.spec.URI)
			if err != nil {
				panic(input.ConfigError.New("failed to parse URI: %s", err))
			}
			bucketName := u.Host
			storePath := u.Path
			var splay bool
			switch u.Scheme {
			case "s3":
				splay = false
			case "s3+splay":
				splay = true
			default:
				panic(input.ConfigError.New("unrecognized scheme: %q", u.Scheme))
			}

			// load keys from env
			// TODO someday URIs should grow smart enough to control this in a more general fashion -- but for now, host ENV is actually pretty feasible and plays easily with others.
			keys, err := s3gof3r.EnvKeys()
			if err != nil {
				panic(S3CredentialsMissingError.Wrap(err))
			}

			// initialize reader from s3!
			getPath := storePath
			if splay {
				getPath = path.Join(storePath, i.spec.Hash)
			}
			s3reader := makeS3reader(bucketName, getPath, keys)
			defer s3reader.Close()

			// prepare decompression as necessary
			reader, err := tar2.Decompress(s3reader)
			if err != nil {
				panic(input.DataSourceUnavailableError.New("could not start decompressing: %s", err))
			}
			tarReader := tar.NewReader(reader)

			// unroll the tar, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			tar2.Extract(tarReader, destinationRoot, bucket, i.hasherFactory)

			// hash whole tree
			actualTreeHash, _ := fshash.Hash(bucket, i.hasherFactory)

			// verify total integrity
			expectedTreeHash, err := base64.URLEncoding.DecodeString(i.spec.Hash)
			if !bytes.Equal(actualTreeHash, expectedTreeHash) {
				done <- input.NewHashMismatchError(i.spec.Hash, base64.URLEncoding.EncodeToString(actualTreeHash))
			}
		}).Catch(input.Error, func(err *errors.Error) {
			done <- err
		}).CatchAll(func(err error) {
			// All errors we emit will be under `input.Error`'s type.
			// Every time we hit this UnknownError path, we should consider it a bug until that error is categorized.
			done <- input.UnknownError.Wrap(err).(*errors.Error)
		}).Done()
	}()
	return done
}

var s3Conf = &s3gof3r.Config{
	Concurrency: 10,
	PartSize:    20 * 1024 * 1024,
	NTry:        10,
	Md5Check:    false,
	Scheme:      "https",
	Client:      s3gof3r.ClientWithTimeout(15 * time.Second),
}

func makeS3reader(bucketName string, path string, keys s3gof3r.Keys) io.ReadCloser {
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	w, _, err := s3.Bucket(bucketName).GetReader(path, s3Conf)
	if err != nil {
		panic(S3Error.Wrap(err))
	}
	return w
}
