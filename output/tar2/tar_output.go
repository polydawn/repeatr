package tar2

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"io"
	"os"
	"time"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/output"
)

const Type = "tar"

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
	done := make(chan output.Report)
	go func() {
		defer close(done)
		try.Do(func() {
			// open output location for writing
			// currently this impl assumes a local file uri
			file, err := os.OpenFile(o.spec.URI, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0755)
			if err != nil {
				panic(output.TargetFilesystemUnavailableIOError(err))
			}
			defer file.Close()

			// walk, fwrite, hash
			o.spec.Hash = Save(file, basePath, o.hasherFactory)

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

// Walks `basePath`, hashing it, pushing the encoded tar to `file`, and returning the final hash.
func Save(file io.Writer, basePath string, hasherFactory func() hash.Hash) string {
	// walk filesystem, copying and accumulating data for integrity check
	bucket := &fshash.MemoryBucket{}
	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()
	if err := walk(basePath, tarWriter, bucket, hasherFactory); err != nil {
		panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
	}

	// hash whole tree
	actualTreeHash, _ := fshash.Hash(bucket, hasherFactory)

	// report
	return base64.URLEncoding.EncodeToString(actualTreeHash)
}

func walk(srcBasePath string, tw *tar.Writer, bucket fshash.Bucket, hasherFactory func() hash.Hash) error {
	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}
		hdr, file := fs.ScanFile(srcBasePath, filenode.Path, filenode.Info)
		// flaten time to seconds.  this tar writer impl doesn't do subsecond precision.
		// the writer will flatten it internally of course, but we need to do it here as well
		// so that the hash and the serial form are describing the same thing.
		hdr.ModTime = hdr.ModTime.Truncate(time.Second)
		wat := tar.Header(hdr) // this line is... we're not gonna talk about this.
		tw.WriteHeader(&wat)
		if file == nil {
			bucket.Record(hdr, nil)
		} else {
			defer file.Close()
			hasher := hasherFactory()
			tee := io.MultiWriter(tw, hasher)
			_, err := io.Copy(tee, file)
			if err != nil {
				return err
			}
			bucket.Record(hdr, hasher.Sum(nil))
		}
		return nil
	}
	return fs.Walk(srcBasePath, preVisit, nil)
}
