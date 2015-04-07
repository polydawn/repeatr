package fshash

import (
	"archive/tar"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/fs"
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
	Record(metadata fs.Metadata, contentHash []byte) // record a file into the bucket
	Iterator() (rootRecord RecordIterator)           // return a treewalk root that does a traversal ordered by path
	Length() int
}

type Record struct {
	// Note: tags are to indicate that this field is serialized, but are a non-functional ornamentation.
	// Serialization code is handcrafted in order to deal with order determinism and does not actually refer to them.

	Metadata    fs.Metadata `json:"m"`
	ContentHash []byte      `json:"h"`
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
	PathCollision is reported when the same file path is submitted to a `Bucket`
	more than once.  (Some formats, for example tarballs, allow the same filename
	to be recorded twice.  We consider this poor behavior since most actual
	filesystems of course will not tolerate this, and also because it begs the
	question of which should be sorted first when creating a deterministic
	hash of the whole tree.)
*/
var PathCollision *errors.ErrorClass = InvalidFilesystem.NewClass("PathCollision")

/*
	MissingTree is reported when iteration over a filled bucket encounters
	a file that has no parent nodes.  I.e., if there's a file path "./a/b",
	and there's no entries for "./a", it's a MissingTree error.
*/
var MissingTree *errors.ErrorClass = InvalidFilesystem.NewClass("MissingTree")

/*
	Node used for the root (Name = ".") path, if one isn't provided.
*/
var DefaultRoot Record

func init() {
	DefaultRoot = Record{
		Metadata: fs.Metadata{
			Name:       ".",
			Typeflag:   tar.TypeDir,
			Mode:       0755,
			ModTime:    def.Epochwhen,
			AccessTime: def.Epochwhen,
			// other fields (uid, gid) have acceptable "zero" values.
		},
		ContentHash: nil,
	}
}

// for sorting
type linesByFilepath []Record

func (a linesByFilepath) Len() int           { return len(a) }
func (a linesByFilepath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a linesByFilepath) Less(i, j int) bool { return a[i].Metadata.Name < a[j].Metadata.Name }
