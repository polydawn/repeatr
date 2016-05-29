package fs

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/lib/fspatch"
)

type Metadata tar.Header

/*
	FileMode returns the permission and mode bits (setuid, setguid, sticky)
	in golang `os.FileMode` format.

	We decline to use `tar.Header.FileInfo()` because the `Mode()` method
	checks *both* the higher mode bits *and* the typeflag, which... seems strange, to
	say the least, since it can result in conflicting information.
	The bits emitted by this method are the only bits that make sense to use anyway --
	see `syscallMode` in stdlib file_posix.go as used in e.g. `os.Mkdir`.

	Note that if you're about to hand a number to a direct linux syscall, you do
	*not* want to use this.  `m.Mode&07777` is safe/correct instead.
*/
func (m Metadata) FileMode() os.FileMode {
	// Set file permission bits.  They're pretty universal.
	mode := os.FileMode(m.Mode).Perm()
	// Map suid, guid, and sticky -- they differ between go os and tar format.
	// Arbitrary constants from tar "spec" -- same as in archive/tar; they're unexported.
	if m.Mode&04000 != 0 {
		mode |= os.ModeSetuid
	}
	if m.Mode&02000 != 0 {
		mode |= os.ModeSetgid
	}
	if m.Mode&01000 != 0 {
		mode |= os.ModeSticky
	}
	return mode
}

/*
	Scan file attributes into a repeatr Metadata struct.  FileInfo
	may be provided if it is already available (this will save a stat call).
	The path is expected to exist (nonexistence is a panicable offense, along
	with all other IO errors).
*/
func ReadMetadata(path string, optional ...os.FileInfo) Metadata {
	// get fileinfo (or reuse, if the caller was kind enough to already have it)
	var fi os.FileInfo
	var err error
	if len(optional) > 0 {
		fi = optional[0]
	} else if len(optional) == 0 {
		fi, err = os.Lstat(path)
		if err != nil {
			// yes, also consider ENOEXIST a problem -- this is
			// never used in a context where a race between the tree
			// walker and the full attribute scan is considered acceptable.
			panic(errors.IOError.Wrap(err))
		}
	} else {
		panic(errors.ProgrammerError.New("optional fileinfo may only be one"))
	}
	// use stdlib tar to lift most of the metadata from fileinfo format
	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}
	// handle extra metadata on the interesting types
	// not handled: hardlinks
	switch hdr.Typeflag {
	case tar.TypeSymlink:
		// readlink needs the file path again  ヽ(´ー｀)ノ
		if hdr.Linkname, err = os.Readlink(path); err != nil {
			panic(errors.IOError.Wrap(err))
		}
	case tar.TypeBlock, tar.TypeChar:
		hdr.Devmajor, hdr.Devminor = fspatch.ReadDev(fi)
	}
	// ctimes are uncontrollable, pave them (╯°□°）╯︵ ┻━┻
	// atimes mutate on read, pave them
	// we don't include these when hashing, but we wouldn't want uncontrolled bytes in a tar output either
	hdr.ChangeTime = def.Epochwhen
	hdr.AccessTime = def.Epochwhen
	return Metadata(*hdr)
}

/*
	Encodes the metadata as a CBOR map -- deterministically; the output is appropriate to feed to a hash and expect consistency.

	We follow the rfc7049 section 3.9 description of "canonical CBOR": namely, map keys are here entered consistently, and in sorted order.
	Except when maps are representing a struct; then it's deterministic order, but specified by (fairly arbitrary) hardcoded choices.

	Errors are panicked.
	(Note that if your writer ever returns an error, the codec library will panic with exactly that.  Yes, including `io.EOF`.)
*/
func (m Metadata) Marshal(out io.Writer) {
	// This doesn't implement `BinaryMarshaller` because we A: don't care and B: are invariably writing to another stream anyway.
	_, enc := codec.GenHelperEncoder(codec.NewEncoder(out, new(codec.CborHandle)))
	// Hack around codec not exporting things very usefully -.-
	const magic_UTF8 = 1
	// Count up how many fields we're about to encode.
	fieldCount := 7
	if m.Linkname != "" {
		fieldCount++
	}
	xattrsLen := len(m.Xattrs)
	if xattrsLen > 0 {
		fieldCount++
	}
	if m.Typeflag == tar.TypeBlock || m.Typeflag == tar.TypeChar {
		fieldCount += 2 // devmajor and devminor will be included for these types
	}
	// Let us begin!
	enc.EncodeMapStart(fieldCount)
	enc.EncodeString(magic_UTF8, "n") // name
	// use basename so hash subtrees are severable
	enc.EncodeString(magic_UTF8, filepath.Base(m.Name))
	// tar format magic numbers for file type  aren't particularly human readable but they're no more or less arbitrary than anyone else's
	enc.EncodeString(magic_UTF8, "t") // type
	enc.EncodeInt(int64(m.Typeflag))
	// mode bits whitelisted, matching `FileMode()`.  basic perms (0777) and setuid/setgid/sticky (07000) only.
	enc.EncodeString(magic_UTF8, "m") // mode -- note this is *not* `os.FileMode`, it's just the perm bits
	enc.EncodeInt(m.Mode & 07777)
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
	if m.Linkname != "" {
		enc.EncodeString(magic_UTF8, "l") // link name (optional)
		enc.EncodeString(magic_UTF8, m.Linkname)
	}
	// disregard uname and gname because they're not very helpful
	// dev numbers only if relevant
	if m.Typeflag == tar.TypeBlock || m.Typeflag == tar.TypeChar {
		enc.EncodeString(magic_UTF8, "dM") // dev major
		enc.EncodeInt(m.Devmajor)
		enc.EncodeString(magic_UTF8, "dm") // dev minor
		enc.EncodeInt(m.Devminor)
	}
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
