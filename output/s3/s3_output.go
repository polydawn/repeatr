package s3

import (
	"bytes"
	"crypto/sha512"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/rlmcpherson/s3gof3r"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/output/tar2"
)

const Type = "s3"

var _ output.Output = &Output{} // interface assertion

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
			u, err := url.Parse(o.spec.URI)
			if err != nil {
				panic(output.ConfigError.New("failed to parse URI: %s", err))
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
				panic(output.ConfigError.New("unrecognized scheme: %q", u.Scheme))
			}

			// load keys from env
			// TODO someday URIs should grow smart enough to control this in a more general fashion -- but for now, host ENV is actually pretty feasible and plays easily with others.
			keys, err := s3gof3r.EnvKeys()
			if err != nil {
				panic(S3CredentialsMissingError.Wrap(err))
			}

			// initialize writer to s3!
			// if the URI indicated splay behavior, first stream data to {$bucketName}:{dirname($storePath)}/.tmp.upload.{basename($storePath)}.{random()};
			// this allows us to start uploading before the final hash is determined and relocate it later.
			// for direct paths, upload into place, because aws already manages atomicity at that scale (and they don't have a rename or copy operation that's free, because uh...?  no time to implement it since 2006, apparently).
			putPath := storePath
			if splay {
				putPath = path.Join(path.Dir(storePath), ".tmp.upload."+path.Base(storePath)+"."+guid.New())
			}
			s3writer := makeS3writer(bucketName, putPath, keys)

			// walk, fwrite, hash
			o.spec.Hash = tar2.Save(s3writer, basePath, o.hasherFactory)

			// flush and check errors on the final write to s3.
			// be advised that this close method does *a lot* of work aside from connection termination.
			// also calling it twice causes the library to wigg out and delete things, i don't even.
			err = s3writer.Close()
			if err != nil {
				panic(S3Error.Wrap(err))
			}

			// if the URI indicated splay behavior, rename the temp filepath to the real one;
			// the upload location is suffixed to make a CA resting place.
			if splay {
				finalPath := path.Join(storePath, o.spec.Hash)
				reloc(bucketName, putPath, finalPath, keys)
			}

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

var s3Conf = &s3gof3r.Config{
	Concurrency: 10,
	PartSize:    20 * 1024 * 1024,
	NTry:        10,
	Md5Check:    false,
	Scheme:      "https",
	Client:      s3gof3r.ClientWithTimeout(15 * time.Second),
}

func makeS3writer(bucketName string, path string, keys s3gof3r.Keys) io.WriteCloser {
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	w, err := s3.Bucket(bucketName).PutWriter(path, nil, s3Conf)
	if err != nil {
		panic(S3Error.Wrap(err))
	}
	return w
}

func reloc(bucketName, oldPath, newPath string, keys s3gof3r.Keys) {
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	bucket := s3.Bucket(bucketName)
	// this is a POST at the bottom, and copies are a PUT.  whee.
	//w, err := s3.Bucket(bucketName).PutWriter(newPath, copyInstruction, s3Conf)
	// So, implement our own aws copy API.
	req, err := http.NewRequest("PUT", "", &bytes.Buffer{})
	if err != nil {
		panic(S3Error.Wrap(err))
	}
	req.URL.Scheme = s3Conf.Scheme
	req.URL.Host = fmt.Sprintf("%s.%s", bucketName, s3.Domain)
	req.URL.Path = path.Clean(fmt.Sprintf("/%s", newPath))
	// Communicate the copy source object with a header.
	// Be advised that if this object doesn't exist, amazon reports that as a 404... yes, a 404 that has nothing to do with the query URI.
	req.Header.Add("x-amz-copy-source", path.Join("/", bucketName, oldPath))
	bucket.Sign(req)
	resp, err := s3Conf.Client.Do(req)
	if err != nil {
		panic(S3Error.Wrap(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		panic(S3Error.Wrap(newRespError(resp)))
	}
	// delete previous location
	if err := bucket.Delete(oldPath); err != nil {
		panic(S3Error.Wrap(err))
	}
}

func newRespError(r *http.Response) *s3gof3r.RespError {
	e := new(s3gof3r.RespError)
	e.StatusCode = r.StatusCode
	b, _ := ioutil.ReadAll(r.Body)
	xml.NewDecoder(bytes.NewReader(b)).Decode(e) // parse error from response
	r.Body.Close()
	return e
}
