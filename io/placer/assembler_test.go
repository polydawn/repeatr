package placer

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

func TestCopyingPlacerCompliance(t *testing.T) {
	t.Skip("none of this is completed implementation yet!")
	CheckPlacementBasics(t, defaultAssembler{Placer: CopyingPlacer}.Assemble)
}

// you probs want to create that assembler with a variety of placers
func CheckPlacementBasics(t *testing.T, assemblerFn integrity.Assembler) {
	Convey("Assembly a series of filesystems should produce a union", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				filefixture.Beta.Create("./material/beta")
				// We're going to try a bunch of tricky things at once:
				// - placement basics, of course
				// - one placement inside another, with a directory that already exists
				// - one placement inside another, with a directory that *doesn't* already exist
				// - reusing one data source in several active locations
				// TODO coverage:
				// - failure path: placement that overlaps a file somewhere
				// - everything about changes and ensuring they're isolated... deserves a whole battery

				assembly := assemblerFn("./assembled", []integrity.AssemblyPart{
					{TargetPath: "/", SourcePath: "./material/alpha"},
					{TargetPath: "/a", SourcePath: "./material/beta"},
					{TargetPath: "/d/d/d", SourcePath: "./material/beta"},
				})

				scan := filefixture.Scan("./assembled")
				So(scan.Describe(filefixture.CompareDefaults), ShouldEqual,
					filefixture.Fixture{Files: []filefixture.FixtureFile{
						{fs.Metadata{Name: ".", Mode: 0755, ModTime: time.Unix(1000, 2000)}, nil}, // even though the basedir was made by the assembler, this should have the rootfs's properties overlayed onto it
						{fs.Metadata{Name: "./a"}, nil},                                           // this one's mode and times should be overlayed by the second mount
						{fs.Metadata{Name: "./a/1"}, []byte{}},
						{fs.Metadata{Name: "./a/2"}, []byte{}},
						{fs.Metadata{Name: "./a/3"}, []byte{}},
						{fs.Metadata{Name: "./b", Mode: 0750, ModTime: time.Unix(5000, 2000)}, nil},
						{fs.Metadata{Name: "./b/c", Mode: 0664, ModTime: time.Unix(7000, 2000)}, []byte("zyx")},
						{fs.Metadata{Name: "./d"}, nil}, // these should have been manifested by the assembler
						{fs.Metadata{Name: "./d/d"}, nil},
						{fs.Metadata{Name: "./d/d/d"}, nil},
						{fs.Metadata{Name: "./d/d/d/1"}, []byte{}},
						{fs.Metadata{Name: "./d/d/d/2"}, []byte{}},
						{fs.Metadata{Name: "./d/d/d/3"}, []byte{}},
					}}.Defaults().Describe(filefixture.CompareDefaults))

				assembly.Teardown()
			}),
		),
	)
}
