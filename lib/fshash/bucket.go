package fshash

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/lib/treewalk"
)

/*
	Bucket keeps hashes of file content and the set of metadata per file and dir.
	This is to make it possible to range over the filesystem out of order and
	construct a total hash of the system in order later.

	Currently this just has an in-memory implementation, but something backed by
	e.g. boltdb for really large file trees would also make sense.
*/
type Bucket interface {
	Record(metadata Metadata, contentHash []byte)
	Iterator() (rootRecord RecordIterator)
}

type Record struct {
	Metadata    Metadata `json:"m"`
	ContentHash []byte   `json:"h"`
}

/*
	RecordIterator is used for walking Bucket contents in hash-ready order.
	It's specified separately from Record for three reasons: There will be many
	Record objects, and so they should be small (the iterators tend to require
	at least another two words of memory); and the Record objects are
	serializable the same way for all implementations of Bucket (the iterators
	may work differently depending on how data is heaped in the Bucket impl).
*/
type RecordIterator interface {
	treewalk.Node
	Record() Record
}

var InvalidFilesystem *errors.ErrorClass = errors.NewClass("InvalidFilesystem")

/*
	FileCollision is reported when the same file path is submitted to a `Bucket`
	more than once.  (Some formats, for example tarballs, allow the same filename
	to be recorded twice.  We consider this poor behavior since most actual
	filesystems of course will not tolerate this, and also because it begs the
	question of which should be sorted first when creating a deterministic
	hash of the whole tree.)
*/
var FileCollision *errors.ErrorClass = InvalidFilesystem.NewClass("FileCollision")

/*
	MissingTree is reported when iteration over a filled bucket encounters
	a file that has no parent nodes.  I.e., if there's a file path "./a/b",
	and there's no entries for "./a", it's a MissingTree error.
*/
var MissingTree *errors.ErrorClass = InvalidFilesystem.NewClass("MissingTree")

// for sorting
type linesByFilepath []Record

func (a linesByFilepath) Len() int           { return len(a) }
func (a linesByFilepath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a linesByFilepath) Less(i, j int) bool { return a[i].Metadata.Name < a[j].Metadata.Name }
