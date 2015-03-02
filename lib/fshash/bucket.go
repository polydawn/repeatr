package fshash

import (
	"archive/tar"

	"github.com/spacemonkeygo/errors"
)

type Metadata tar.Header

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
}

func (b *MemoryBucket) Record(metadata Metadata, contentHash []byte) {

}
