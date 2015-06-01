package tar2

import (
	"archive/tar"
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"io"
	"os"
	"path"
	"strings"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/lib/treewalk"
)

const Type = "tar"

var _ input.Input = &Input{} // interface assertion

type Input struct {
	spec          def.Input
	hasherFactory func() hash.Hash
}

func New(spec def.Input) input.Input {
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
			// daemon uid and gid are fine for now (they're always 0);
			// forcing a constant time happens along with all the other dirs, since
			// the bucket normalizes to include a root attribute set if the tar doesn't have one.

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
			Extract(tarReader, destinationRoot, bucket, i.hasherFactory)

			// bucket processing may have created a root node if missing.  if so, we need to apply its props.
			fs.PlaceFile(destinationRoot, bucket.Root().Metadata, nil)

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

func Extract(tr *tar.Reader, destBasePath string, bucket fshash.Bucket, hasherFactory func() hash.Hash) error {
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
		// - names must be clean, relative dot-slash prefixed, and dirs slash-suffixed
		// - times should never be go's zero value; replace those with epoch
		// Note that names at this point should be handled by `path` (not `filepath`; these are canonical form for feed to hashing)
		hdr.Name = path.Clean(hdr.Name)
		if strings.HasPrefix(hdr.Name, "../") {
			panic(input.DataSourceUnavailableError.New("corrupt tar: paths that use '../' to leave the base dir are invalid"))
		}
		if hdr.Name != "." {
			hdr.Name = "./" + hdr.Name
		}
		if hdr.ModTime.IsZero() {
			hdr.ModTime = def.Epochwhen
		}
		if hdr.AccessTime.IsZero() {
			hdr.AccessTime = def.Epochwhen
		}
		// place the file
		switch hdr.Typeflag {
		case tar.TypeReg:
			reader := &flak.HashingReader{tr, hasherFactory()}
			fs.PlaceFile(destBasePath, hdr, reader)
			bucket.Record(hdr, reader.Hasher.Sum(nil))
		case tar.TypeDir:
			hdr.Name += "/"
			fallthrough
		default:
			fs.PlaceFile(destBasePath, hdr, nil)
			bucket.Record(hdr, nil)
		}
	}
	// cleanup dir times with a post-order traversal over the bucket
	if err := treewalk.Walk(bucket.Iterator(), nil, func(node treewalk.Node) error {
		record := node.(fshash.RecordIterator).Record()
		if record.Metadata.Typeflag == tar.TypeDir {
			fs.PlaceDirTime(destBasePath, record.Metadata)
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return nil
}
