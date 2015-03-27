package filefixture

import (
	"archive/tar"
	"bytes"
	"strings"
	"time"

	"polydawn.net/repeatr/lib/fs"
)

type FixtureFile struct {
	Metadata fs.Metadata
	Body     []byte
}

type Fixture struct {
	Name  string
	Files []FixtureFile
}

func defaults(f FixtureFile) FixtureFile {
	if f.Metadata.Typeflag == '\x00' {
		if f.Body == nil {
			f.Metadata.Typeflag = tar.TypeDir
		} else {
			f.Metadata.Typeflag = tar.TypeReg
		}
	}
	if f.Metadata.Mode == 0 {
		switch f.Metadata.Typeflag {
		case tar.TypeDir:
			f.Metadata.Mode = 0755
		default:
			f.Metadata.Mode = 0644
		}
	}
	if f.Metadata.Uid == 0 {
		f.Metadata.Uid = 10000
	}
	if f.Metadata.Gid == 0 {
		f.Metadata.Gid = 10000
	}
	if f.Metadata.ModTime.IsZero() {
		f.Metadata.ModTime = time.Unix(0, 0).UTC()
	}
	if f.Metadata.AccessTime.IsZero() {
		f.Metadata.AccessTime = time.Unix(0, 0).UTC()
	}
	return f
}

/*
	Create files described by the fixtures on the real filesystem path given.
*/
func (ffs Fixture) Create(basePath string) {
	for _, f := range ffs.Files {
		fs.PlaceFile(basePath, f.Metadata, bytes.NewBuffer(f.Body))
	}
}

/*
	Scan a real filesystem and see it as fixture file descriptions.
	Usually used as a prelude to a pair of `Describe` calls followed by
	an equality assertion.

	Note that this loads all file bodies into memory at once, so it
	is not wise to use on large filesystems.

	Result will be sorted by filename, as per usual.
*/
func Scan(basePath string) Fixture {
	return Fixture{} // TODO
}

/*
	Produces a string where every line describes a one entry in the
	set of file descriptions.  This is useful for handing into "ShouldResemble"
	assertions, which will not only pass/fail but do character diffs which
	effectively cover the whole structure in one shot.


*/
func (ffs Fixture) Describe() string {
	lines := make([]string, len(ffs.Files))
	for i, f := range ffs.Files {
		lines[i] = f.Describe()
	}
	return strings.Join(lines, "\n")
}

/*
	As per `FixtureFiles.Describe`, but this is for a single entry.
*/
func (ff FixtureFile) Describe() string {
	return "" // TODO
}
