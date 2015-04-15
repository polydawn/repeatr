package filefixture

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
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

type filesByPath []FixtureFile

func (a filesByPath) Len() int           { return len(a) }
func (a filesByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a filesByPath) Less(i, j int) bool { return a[i].Metadata.Name < a[j].Metadata.Name }

func (f Fixture) defaults() Fixture {
	for i, ff := range f.Files {
		f.Files[i] = defaults(ff)
	}
	return f
}

func defaults(f FixtureFile) FixtureFile {
	if f.Metadata.Typeflag == '\x00' {
		if f.Body == nil {
			if f.Metadata.Linkname != "" {
				f.Metadata.Typeflag = tar.TypeSymlink
			} else {
				f.Metadata.Typeflag = tar.TypeDir
			}
		} else {
			f.Metadata.Typeflag = tar.TypeReg
		}
	}
	switch f.Metadata.Typeflag {
	case tar.TypeDir:
		if !strings.HasSuffix(f.Metadata.Name, "/") {
			f.Metadata.Name += "/"
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
		f.Metadata.ModTime = def.Epochwhen
	}
	if f.Metadata.AccessTime.IsZero() {
		f.Metadata.AccessTime = def.Epochwhen
	}
	return f
}

/*
	Create files described by the fixtures on the real filesystem path given.
*/
func (ffs Fixture) Create(basePath string) {
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}
	for _, f := range ffs.Files {
		fs.PlaceFile(basePath, f.Metadata, bytes.NewBuffer(f.Body))
	}
	// re-do time enforcement... in reverse order, so we cover our own tracks
	for i := len(ffs.Files) - 1; i >= 0; i-- {
		f := ffs.Files[i]
		if f.Metadata.Typeflag == tar.TypeDir {
			fs.PlaceDirTime(basePath, f.Metadata)
		}
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
	ffs := Fixture{fmt.Sprintf("Scan of %q", basePath), nil}
	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}
		hdr, file := fs.ScanFile(basePath, filenode.Path)
		var body []byte
		if file != nil {
			defer file.Close()
			var err error
			body, err = ioutil.ReadAll(file)
			if err != nil {
				return err
			}
		}
		ffs.Files = append(ffs.Files, FixtureFile{hdr, body})
		return nil
	}
	if err := fs.Walk(basePath, preVisit, nil); err != nil {
		panic(err)
	}
	sort.Sort(filesByPath(ffs.Files))
	return ffs
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
		{"Perms:%q", (map[bool]interface{}{true: "-", false: ff.Metadata.FileMode()})[opts&ComparePerms == 0]},
		{"Mtime:%q", (map[bool]interface{}{true: "-", false: ff.Metadata.ModTime.UTC()})[opts&CompareMtime == 0]},
		{"Atime:%q", (map[bool]interface{}{true: "-", false: ff.Metadata.AccessTime.UTC()})[opts&CompareAtime == 0]},
		{"Uid:%d", (map[bool]interface{}{true: "-", false: ff.Metadata.Uid})[opts&CompareUid == 0]},
		{"Gid:%d", (map[bool]interface{}{true: "-", false: ff.Metadata.Gid})[opts&CompareGid == 0]},
		{"DM:%d", ff.Metadata.Devmajor},
		{"Dm:%d", ff.Metadata.Devminor},
		{"Link:%q", ff.Metadata.Linkname},
		{"Size:%d", (map[bool]interface{}{true: "-", false: ff.Metadata.Size})[opts&CompareSize == 0]},
		{"Body:%q", (map[bool]interface{}{true: "-", false: ff.Body})[opts&CompareBody == 0]},
	}
	var pattern string
	var values []interface{}
	for _, part := range parts {
		pattern += "\t" + part.Key
		values = append(values, part.Value)
	}
	return fmt.Sprintf(pattern, values...)
}
