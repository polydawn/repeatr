package filefixture

import (
	"archive/tar"
	"bytes"
	"fmt"
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

type ComparisonOptions uint32

const (
	ComparePerms = ComparisonOptions(0001)
	CompareMtime = ComparisonOptions(0002)
	CompareAtime = ComparisonOptions(0004)
	CompareUid   = ComparisonOptions(0010)
	CompareGid   = ComparisonOptions(0020)
	CompareSize  = ComparisonOptions(0040)
	CompareBody  = ComparisonOptions(0100)

	CompareDefaults = ComparePerms | CompareMtime | CompareUid | CompareGid |
		CompareSize | CompareBody
	CompareAll = CompareDefaults | CompareAtime
)

func (f Fixture) defaults() Fixture {
	for i, ff := range f.Files {
		f.Files[i] = defaults(ff)
	}
	return f
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
	if f.Metadata.Size == 0 {
		f.Metadata.Size = int64(len(f.Body))
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
func (ffs Fixture) Describe(opts ComparisonOptions) string {
	lines := make([]string, len(ffs.Files))
	for i, f := range ffs.Files {
		lines[i] = f.Describe(opts)
	}
	return strings.Join(lines, "\n")
}

/*
	As per `FixtureFiles.Describe`, but this is for a single entry.
*/
func (ff FixtureFile) Describe(opts ComparisonOptions) string {
	parts := []struct {
		Key   string
		Value interface{}
	}{
		{"Name:%q", ff.Metadata.Name},
		{"Type:%q", ff.Metadata.Typeflag},
		{"Perms:%q", ff.Metadata.FileMode()},
		{"Mtime:%q", ff.Metadata.ModTime},
		{"Atime:%q", ff.Metadata.AccessTime},
		{"Uid:%d", ff.Metadata.Uid},
		{"Gid:%d", ff.Metadata.Gid},
		{"DM:%d", ff.Metadata.Devmajor},
		{"Dm:%d", ff.Metadata.Devminor},
		{"Link:%q", ff.Metadata.Linkname},
		{"Size:%d", ff.Metadata.Size},
		{"Body:%q", ff.Body},
	}
	// my kingdom for a ternary operator
	if opts&ComparePerms == 0 {
		parts[2].Value = "-"
	}
	if opts&CompareMtime == 0 {
		parts[3].Value = "-"
	}
	if opts&CompareAtime == 0 {
		parts[4].Value = "-"
	}
	if opts&CompareUid == 0 {
		parts[5].Value = "-"
	}
	if opts&CompareGid == 0 {
		parts[6].Value = "-"
	}
	if opts&CompareSize == 0 {
		parts[10].Value = "-"
	}
	if opts&CompareBody == 0 {
		parts[11].Value = "-"
	}
	var pattern string
	var values []interface{}
	for _, part := range parts {
		pattern += "\t" + part.Key
		values = append(values, part.Value)
	}
	return fmt.Sprintf(pattern, values...)
}
