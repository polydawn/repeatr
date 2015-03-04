package fshash

import (
	"archive/tar"
	"os"

	"github.com/spacemonkeygo/errors"
)

type Metadata tar.Header

func ReadMetadata(path string, optional ...os.FileInfo) Metadata {
	var fi os.FileInfo
	var err error
	if len(optional) > 0 {
		fi = optional[0]
	} else {
		fi, err = os.Lstat(path)
		if err != nil {
			// also consider ENOEXIST a problem; this function is mostly
			// used in testing where we really expect that path to exist.
			panic(errors.IOError.Wrap(err))
		}
	}
	// readlink needs the file path again  ヽ(´ー｀)ノ
	var link string
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		if link, err = os.Readlink(path); err != nil {
			panic(errors.IOError.Wrap(err))
		}
	}
	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}
	return Metadata(*hdr)
}

func (m Metadata) MarshalBinary() ([]byte, error) {
	// TODO: carefully.  maybe use cbor, but make sure order is consistent and encoding unambiguous.
	// TODO: switch name to basename, so hash subtrees are severable
	return nil, nil
}

/*
	Bucket keeps hashes of file content and the set of metadata per file and dir.
	This is to make it possible to range over the filesystem out of order and
	construct a total hash of the system in order later.

	Currently this just has an in-memory implementation, but something backed by
	e.g. boltdb for really large file trees would also make sense.
*/
type Bucket interface {
	Record(metadata Metadata, contentHash []byte)
}

/*
	FileCollision is reported when the same file path is submitted to a `Bucket`
	more than once.  (Some formats, for example tarballs, allow the same filename
	to be recorded twice.  We consider this poor behavior since most actual
	filesystems of course will not tolerate this, and also because it begs the
	question of which should be sorted first when creating a deterministic
	hash of the whole tree.)
*/
var FileCollision *errors.ErrorClass = errors.NewClass("FileCollision")

var _ Bucket = &MemoryBucket{}

type MemoryBucket struct {
	// my kingdom for a red-black tree or other sane sorted map implementation
	lines []line
}

type line struct {
	metadata    Metadata
	contentHash []byte
}

type linesByFilepath []line

func (a linesByFilepath) Len() int           { return len(a) }
func (a linesByFilepath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a linesByFilepath) Less(i, j int) bool { return a[i].metadata.Name < a[j].metadata.Name }

func (b *MemoryBucket) Record(metadata Metadata, contentHash []byte) {
	b.lines = append(b.lines, line{metadata, contentHash})
}
