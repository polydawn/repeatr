package tar2

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"io"
	"os"

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
				done <- output.Report{errors.IOError.Wrap(err).(*errors.Error), def.Output{}}
			}
			defer file.Close()

			// walk filesystem, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			tarWriter := tar.NewWriter(file)
			defer tarWriter.Close()
			if err := walk(basePath, tarWriter, bucket, o.hasherFactory); err != nil {
				done <- output.Report{err.(*errors.Error), def.Output{}}
				return
			}

			// hash whole tree
			actualTreeHash, _ := fshash.Hash(bucket, o.hasherFactory)

			// report
			o.spec.Hash = base64.URLEncoding.EncodeToString(actualTreeHash)
			done <- output.Report{nil, o.spec}
		}).Catch(output.Error, func(err *errors.Error) {
			done <- output.Report{err, def.Output{}}
		}).CatchAll(func(err error) {
			// All errors we emit will be under `output.Error`'s type.
			// Every time we hit this UnknownError path, we should consider it a bug until that error is categorized.
			done <- output.Report{output.UnknownError.Wrap(err).(*errors.Error), def.Output{}}
		}).Done()
	}()
	return done
}

func walk(srcBasePath string, tw *tar.Writer, bucket fshash.Bucket, hasherFactory func() hash.Hash) error {
	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}
		hdr, file := fs.ScanFile(srcBasePath, filenode.Path, filenode.Info)
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
