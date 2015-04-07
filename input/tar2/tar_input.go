package tar2

import (
	"archive/tar"
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"io"
	"os"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/lib/fspatch"
)

const Type = "tar"

var _ input.Input = &Input{} // interface assertion

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
		panic(input.DataSourceUnavailableError.New("Input URI %q must be a tar file", spec.URI))
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
			// daemon uid and gid are fine for now but force a constant mtime.
			if err := fspatch.LUtimesNano(destinationRoot, def.Epochwhen, def.Epochwhen); err != nil {
				panic(input.TargetFilesystemUnavailableIOError(err))
			}

			// open the tar file; preparing decompression as necessary
			file, err := os.OpenFile(i.spec.URI, os.O_RDONLY, 0755)
			if err != nil {
				panic(input.DataSourceUnavailableIOError(err))
			}
			defer file.Close()
			reader, err := Decompress(file)
			if err != nil {
				panic(input.DataSourceUnavailableError.New("could not start decompressing: %s", err))
			}
			tarReader := tar.NewReader(reader)

			// unroll the tar, copying and accumulating data for integrity check
			bucket := &fshash.MemoryBucket{}
			walk(tarReader, destinationRoot, bucket, i.hasherFactory)

			// hash whole tree
			actualTreeHash, _ := fshash.Hash(bucket, i.hasherFactory)

			// verify total integrity
			expectedTreeHash, err := base64.URLEncoding.DecodeString(i.spec.Hash)
			if !bytes.Equal(actualTreeHash, expectedTreeHash) {
				done <- input.InputHashMismatchError.New("expected hash %q, got %q", i.spec.Hash, base64.URLEncoding.EncodeToString(actualTreeHash))
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

func walk(tr *tar.Reader, destBasePath string, bucket fshash.Bucket, hasherFactory func() hash.Hash) error {
	for {
		thdr, err := tr.Next()
		if err == io.EOF {
			break // end of archive
		}
		if err != nil {
			panic(input.DataSourceUnavailableError.New("corrupt tar: %s", err))
		}
		hdr := fs.Metadata(*thdr)
		// filter/sanify values:
		// times should never be go's zero value; replace those with epoch
		if hdr.ModTime.IsZero() {
			hdr.ModTime = def.Epochwhen
		}
		if hdr.AccessTime.IsZero() {
			hdr.AccessTime = def.Epochwhen
		}
		// place the file
		switch hdr.Typeflag {
		case tar.TypeReg:
			reader := &hashingReader{
				r:      tr,
				hasher: hasherFactory(),
			}
			fs.PlaceFile(destBasePath, hdr, reader)
			bucket.Record(hdr, reader.hasher.Sum(nil))
		default:
			fs.PlaceFile(destBasePath, hdr, tr)
			bucket.Record(hdr, nil)
			// TODO fixup dir times afterwards
		}
	}
	return nil
}

/*
	Proxies a reader, hashing the stream as it's read.
	(This is useful if using `io.Copy` to move bytes from a reader to
	a writer, and you want to use that goroutine to power the hashing as
	well but replacing the writer with a multiwriter is out of bounds.)
*/
type hashingReader struct {
	r      io.Reader
	hasher hash.Hash
}

func (r *hashingReader) Read(b []byte) (int, error) {
	n, err := r.r.Read(b)
	r.hasher.Write(b[:n])
	return n, err
}
