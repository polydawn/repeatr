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
	Convey("Copying placers make data appear into place", t, func() {
		CheckAssemblerGetsDataIntoPlace(defaultAssembler{Placer: CopyingPlacer}.Assemble)
	})
}

func TestBindPlacerCompliance(t *testing.T) {
	Convey("Bind placers make data appear into place", t, func() {
		CheckAssemblerGetsDataIntoPlace(defaultAssembler{Placer: BindPlacer}.Assemble)
	})
}

// you probs want to create that assembler with a variety of placers
func CheckAssemblerGetsDataIntoPlace(assemblerFn integrity.Assembler) {
	Convey("Assembly with just a root fs works",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				assembleAndScan(
					assemblerFn,
					[]integrity.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha"},
					},
					filefixture.Alpha,
				)
			}),
		),
	)

	Convey("Assembly with one placement into an existing dir works",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				filefixture.Beta.Create("./material/beta")
				assembleAndScan(
					assemblerFn,
					[]integrity.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha"},
						{TargetPath: "/a", SourcePath: "./material/beta"},
					},
					filefixture.ConjoinFixtures([]filefixture.FixtureAssemblyPart{
						{TargetPath: "/", Fixture: filefixture.Alpha},
						{TargetPath: "/a", Fixture: filefixture.Beta},
					}),
				)
			}),
		),
	)

	Convey("Assembly with one placement into an implicitly-created dir works",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				filefixture.Beta.Create("./material/beta")
				assembleAndScan(
					assemblerFn,
					[]integrity.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha"},
						{TargetPath: "/q", SourcePath: "./material/beta"},
					},
					filefixture.ConjoinFixtures([]filefixture.FixtureAssemblyPart{
						{TargetPath: "/", Fixture: filefixture.Alpha},
						{TargetPath: "/q", Fixture: filefixture.Beta},
					}),
				)
			}),
		),
	)

	Convey("Assembly with overlapping placements shows only top layer",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				filefixture.Beta.Create("./material/beta")
				// TODO
			}),
		),
	)

	Convey("Assembly using the same base twice works",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				filefixture.Beta.Create("./material/beta")
				assembleAndScan(
					assemblerFn,
					[]integrity.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha"},
						{TargetPath: "/q", SourcePath: "./material/beta"},
						{TargetPath: "/w", SourcePath: "./material/beta"},
					},
					filefixture.ConjoinFixtures([]filefixture.FixtureAssemblyPart{
						{TargetPath: "/", Fixture: filefixture.Alpha},
						{TargetPath: "/q", Fixture: filefixture.Beta},
						{TargetPath: "/w", Fixture: filefixture.Beta},
					}),
				)
			}),
		),
	)

	Convey("Assembly with implicitly created deep dirs works",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				filefixture.Beta.Create("./material/beta")
				assembleAndScan(
					assemblerFn,
					[]integrity.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha"},
						{TargetPath: "/a", SourcePath: "./material/beta"},
						{TargetPath: "/d/d/d", SourcePath: "./material/beta"},
					},
					filefixture.Fixture{Files: []filefixture.FixtureFile{
						{fs.Metadata{Name: ".", Mode: 0755, ModTime: time.Unix(1000, 2000)}, nil}, // even though the basedir was made by the assembler, this should have the rootfs's properties overlayed onto it
						{fs.Metadata{Name: "./a"}, nil},                                           // this one's mode and times should be overlayed by the second mount
						{fs.Metadata{Name: "./a/1"}, []byte{}},
						{fs.Metadata{Name: "./a/2"}, []byte{}},
						{fs.Metadata{Name: "./a/3"}, []byte{}},
						{fs.Metadata{Name: "./b", Mode: 0750, ModTime: time.Unix(5000, 2000)}, nil},
						{fs.Metadata{Name: "./b/c", Mode: 0664, ModTime: time.Unix(7000, 2000)}, []byte("zyx")},
						{fs.Metadata{Name: "./d", Uid: -1, Gid: -1}, nil}, // these should have been manifested by the assembler
						{fs.Metadata{Name: "./d/d", Uid: -1, Gid: -1}, nil},
						{fs.Metadata{Name: "./d/d/d"}, nil},
						{fs.Metadata{Name: "./d/d/d/1"}, []byte{}},
						{fs.Metadata{Name: "./d/d/d/2"}, []byte{}},
						{fs.Metadata{Name: "./d/d/d/3"}, []byte{}},
					}}.Defaults(),
				)
			}),
		),
	)

	// additional coverage todos:
	// - failure path: placement that overlaps a file somewhere
	// - everything about changes and ensuring they're isolated... deserves a whole battery
}

func assembleAndScan(assemblerFn integrity.Assembler, parts []integrity.AssemblyPart, expected filefixture.Fixture) {
	Convey("Assembly should not blow up", FailureContinues, func() {
		var assembly integrity.Assembly
		So(func() {
			assembly = assemblerFn("./assembled", parts)
		}, ShouldNotPanic)

		Convey("Filesystem should scan as the expected union", func() {
			scan := filefixture.Scan("./assembled")
			So(scan.Describe(filefixture.CompareDefaults), ShouldEqual, expected.Describe(filefixture.CompareDefaults))

			if assembly != nil {
				// conditional only because we may have continued moving after an error earlier.
				assembly.Teardown()
			}
		})
	})
}

func CheckAssemblerSequestersData(assemblerFn integrity.Assembler) {
	// TODO outputs will rely on these behaviors
}

func CheckAssemblerRespectsReadonly(assemblerFn integrity.Assembler) {
	// TODO
}
