package fshash

import (
	"archive/tar"
	"os"
	"sort"
	"strings"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/treewalk"
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
	// ctimes are uncontrollable, pave them (╯°□°）╯︵ ┻━┻
	// atimes mutate on read, pave them
	hdr.ChangeTime = def.Somewhen
	hdr.AccessTime = def.Somewhen
	return Metadata(*hdr)
}

func (m Metadata) MarshalBinary() ([]byte, error) {
	// TODO: carefully.  maybe use cbor, but make sure order is consistent and encoding unambiguous.
	// TODO: switch name to basename, so hash subtrees are severable
	// disregard atime and ctime because they are almost and completely unusable, respectively
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

var _ Bucket = &MemoryBucket{}

type MemoryBucket struct {
	// my kingdom for a red-black tree or other sane sorted map implementation
	lines []Record
}

func (b *MemoryBucket) Record(metadata Metadata, contentHash []byte) {
	b.lines = append(b.lines, Record{metadata, contentHash})
}

func (b *MemoryBucket) Iterator() RecordIterator {
	sort.Sort(linesByFilepath(b.lines))
	// TODO: check for rootedness
	var that int
	return &memoryBucketIterator{b.lines, 0, &that}
}

type memoryBucketIterator struct {
	lines []Record
	this  int  // pretending a linear structure is a tree is weird.
	that  *int // this is the last child walked.
}

func (i *memoryBucketIterator) NextChild() treewalk.Node {
	// is the next one still a child?
	next := *i.that + 1
	if next >= len(i.lines) {
		return nil
	}
	if strings.HasPrefix(i.lines[next].Metadata.Name, i.lines[i.this].Metadata.Name+"/") {
		*i.that = next
		// TODO: check for missing trees
		// TODO: check for repeated names
		return &memoryBucketIterator{i.lines, *i.that, i.that}
	}
	return nil
}

func (i memoryBucketIterator) Record() Record {
	return i.lines[i.this]
}

// for sorting
type linesByFilepath []Record

func (a linesByFilepath) Len() int           { return len(a) }
func (a linesByFilepath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a linesByFilepath) Less(i, j int) bool { return a[i].Metadata.Name < a[j].Metadata.Name }
