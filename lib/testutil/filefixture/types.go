package filefixture

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
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
	ComparePerms     = ComparisonOptions(00001)
	CompareMtime     = ComparisonOptions(00002)
	CompareAtime     = ComparisonOptions(00004)
	CompareSubsecond = ComparisonOptions(01000) // modifies atime and mtime
	CompareUid       = ComparisonOptions(00010)
	CompareGid       = ComparisonOptions(00020)
	CompareSize      = ComparisonOptions(00040)
	CompareBody      = ComparisonOptions(00100)

	CompareDefaults = ComparePerms | CompareMtime | CompareSubsecond | CompareUid | CompareGid |
		CompareSize | CompareBody
	CompareAll = CompareDefaults | CompareAtime
)

type filesByPath []FixtureFile

func (a filesByPath) Len() int           { return len(a) }
func (a filesByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a filesByPath) Less(i, j int) bool { return a[i].Metadata.Name < a[j].Metadata.Name }

func (f Fixture) Defaults() Fixture {
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
	// it's reallllly unfortunate when golang's zero values overlap with your actual data domain
	switch f.Metadata.Uid {
	case -1:
		f.Metadata.Uid = 0
	case 0:
		f.Metadata.Uid = 10000
	}
	switch f.Metadata.Gid {
	case -1:
		f.Metadata.Gid = 0
	case 0:
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

type FixtureAssemblyPart struct {
	TargetPath string
	Fixture    Fixture
}

// you know, at some point these tiny little variations in structs that i keep having to define swap methods for... get rather old
type fixtureAssemblyPartsByPath []FixtureAssemblyPart

func (a fixtureAssemblyPartsByPath) Len() int           { return len(a) }
func (a fixtureAssemblyPartsByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a fixtureAssemblyPartsByPath) Less(i, j int) bool { return a[i].TargetPath < a[j].TargetPath }

func ConjoinFixtures(fixtureParts []FixtureAssemblyPart) (result Fixture) {
	sort.Sort(fixtureAssemblyPartsByPath(fixtureParts))
	for _, fixturePart := range fixtureParts {
		landingPath := "." + path.Clean(fixturePart.TargetPath)
		// do a full new result array.  easiest to filter this way.
		prevResultFiles := result.Files
		result.Files = make([]FixtureFile, 0, len(prevResultFiles)+len(fixturePart.Fixture.Files)+3) // 3 as a fudge factor for implicit mkdirs
		// check for implicit mkdirs
		// TODO
		// check for blowing away
		for _, file := range prevResultFiles {
			if !strings.HasPrefix(file.Metadata.Name, landingPath) {
				result.Files = append(result.Files, file)
			}
		}
		// append
		for _, file := range fixturePart.Fixture.Files {
			file.Metadata.Name = fshash.Normalize(path.Join(landingPath, file.Metadata.Name), file.Metadata.Typeflag == tar.TypeDir)
			result.Files = append(result.Files, file)
		}
	}
	sort.Sort(filesByPath(result.Files))
	return
}

/*
	Create files described by the fixtures on the real filesystem path given.
*/
func (ffs Fixture) Create(basePath string) {
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}
	if err := os.MkdirAll(basePath, 0755); err != nil {
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
	if opts&CompareSubsecond == 0 {
		ff.Metadata.ModTime = ff.Metadata.ModTime.Truncate(time.Second)
		ff.Metadata.AccessTime = ff.Metadata.AccessTime.Truncate(time.Second)
	}
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
