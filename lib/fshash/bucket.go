package fshash

import (
	"archive/tar"
	"io"
	"os"
	"sort"
	"time"

	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"
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

func (m Metadata) Marshal(out io.Writer) {
	// Encodes the metadata as a CBOR map -- deterministically; the output is appropriate to feed to a hash and expect consistency.
	// We follow the rfc7049 section 3.9 description of "canonical CBOR": namely, map keys are here entered consistently, and in sorted order.
	// Except when maps are representing a struct; then it's deterministic order, but specified by (fairly arbitrary) hardcoded choices.
	// This doesn't implement `BinaryMarshaller` because we A: don't care and B: are invariably writing to another stream anyway.
	// Note that if your writer ever returns an error, the codec library will panic with exactly that.  Yes, including `io.EOF`.
	_, enc := codec.GenHelperEncoder(codec.NewEncoder(out, new(codec.CborHandle)))
	// Hack around codec not exporting things very usefully -.-
	const magic_UTF8 = 1
	// Count up how many fields we're about to encode.
	fieldCount := 6
	if m.Linkname != "" {
		fieldCount++
	}
	xattrsLen := len(m.Xattrs)
	if xattrsLen > 0 {
		fieldCount++
	}
	// Let us begin!
	enc.EncodeMapStart(fieldCount)
	enc.EncodeString(magic_UTF8, "n")    // name
	enc.EncodeString(magic_UTF8, m.Name) // REVIEW: consider switch name back to basename, so hash subtrees are severable -- but this would require the hashing walker to actually you know encode and hash things as a tree... yeahhhhh it should probably do that.
	enc.EncodeString(magic_UTF8, "m")    // mode
	enc.EncodeInt(m.Mode)
	enc.EncodeString(magic_UTF8, "u") // uid
	enc.EncodeInt(int64(m.Uid))
	enc.EncodeString(magic_UTF8, "g") // gid
	enc.EncodeInt(int64(m.Gid))
	// skipped size because that's fairly redundant (and we never use hashes that are subject to length extension)
	if m.ModTime.IsZero() { // pretend that golang's zero time is unix epoch
		m.ModTime = time.Unix(0, 0)
	}
	enc.EncodeString(magic_UTF8, "tm") // modified time
	enc.EncodeInt(m.ModTime.Unix())
	enc.EncodeString(magic_UTF8, "tmn") // modified time, nano component
	enc.EncodeInt(int64(m.ModTime.Nanosecond()))
	// disregard atime and ctime because they are almost and completely unusable, respectively (change on read and unsettable)
	// skipped Typeflag because that's pretty redundant with the mode bits
	if m.Linkname != "" {
		enc.EncodeString(magic_UTF8, "l") // link name (optional)
		enc.EncodeString(magic_UTF8, m.Linkname)
	}
	// disregard uname and gname because they're not very helpful
	// disregard dev numbers -- not because we should, but because golang stdlib tar isn't reading them at the moment anyway, so there's More Work to be done for these
	// Xattrs are a mite more complicated because we have to handle unknown keys:
	if xattrsLen > 0 {
		enc.EncodeString(magic_UTF8, "x")
		sorted := make([]stringPair, 0, xattrsLen)
		for k, v := range m.Xattrs {
			sorted = append(sorted, stringPair{k, v})
		}
		sort.Sort(sortableStringPair(sorted))
		enc.EncodeMapStart(xattrsLen)
		for _, line := range sorted {
			enc.EncodeString(magic_UTF8, line.a)
			enc.EncodeString(magic_UTF8, line.b)
		}
	}
	// There is no map-end to encode in cbor since we used the fixed-length map.  We're done.
}

type stringPair struct{ a, b string }
type sortableStringPair []stringPair

func (p sortableStringPair) Len() int           { return len(p) }
func (p sortableStringPair) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p sortableStringPair) Less(i, j int) bool { return p[i].a < p[j].a }

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
