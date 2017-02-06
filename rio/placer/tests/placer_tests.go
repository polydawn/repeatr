package tests

import (
	"os"
	"syscall"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/lib/testutil"
	"go.polydawn.net/repeatr/lib/testutil/filefixture"
	"go.polydawn.net/repeatr/rio"
)

func CheckAssemblerGetsDataIntoPlace(assemblerFn rio.Assembler) {
	Convey("Assembly with just a root fs works",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				assembleAndScan(
					assemblerFn,
					[]rio.AssemblyPart{
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
					[]rio.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha", Writable: true},
						{TargetPath: "/a", SourcePath: "./material/beta", Writable: true},
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
					[]rio.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha", Writable: true},
						{TargetPath: "/q", SourcePath: "./material/beta", Writable: true},
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
				assembleAndScan(
					assemblerFn,
					[]rio.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha", Writable: true},
						// this one's interesting because ./b/c is already a file
						{TargetPath: "/b", SourcePath: "./material/beta", Writable: true},
					},
					filefixture.ConjoinFixtures([]filefixture.FixtureAssemblyPart{
						{TargetPath: "/", Fixture: filefixture.Alpha},
						{TargetPath: "/b", Fixture: filefixture.Beta},
					}),
				)
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
					[]rio.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha", Writable: true},
						{TargetPath: "/q", SourcePath: "./material/beta", Writable: true},
						{TargetPath: "/w", SourcePath: "./material/beta", Writable: true},
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
					[]rio.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha", Writable: true},
						{TargetPath: "/a", SourcePath: "./material/beta", Writable: true},
						{TargetPath: "/d/d/d", SourcePath: "./material/beta", Writable: true},
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

	Convey("Assembly using a file works",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				assembleAndScan(
					assemblerFn,
					[]rio.AssemblyPart{
						{TargetPath: "/", SourcePath: "./material/alpha", Writable: true},
						{TargetPath: "/d/f", SourcePath: "./material/alpha/b/c", Writable: true},
					},
					filefixture.Fixture{Files: []filefixture.FixtureFile{
						{fs.Metadata{Name: ".", Mode: 0755, ModTime: time.Unix(1000, 2000)}, nil},
						{fs.Metadata{Name: "./a", Mode: 01777, ModTime: time.Unix(3000, 2000)}, nil},
						{fs.Metadata{Name: "./b", Mode: 0750, ModTime: time.Unix(5000, 2000)}, nil},
						{fs.Metadata{Name: "./b/c", Mode: 0664, ModTime: time.Unix(7000, 2000)}, []byte("zyx")},
						{fs.Metadata{Name: "./d", Uid: -1, Gid: -1}, nil}, // this should have been manifested by the assembler
						{fs.Metadata{Name: "./d/f", Mode: 0664, ModTime: time.Unix(7000, 2000)}, []byte("zyx")},
					}}.Defaults(),
				)
			}),
		),
	)

	// additional coverage todos:
	// - failure path: placement that overlaps a file somewhere
	// - everything about changes and ensuring they're isolated... deserves a whole battery
}

func assembleAndScan(assemblerFn rio.Assembler, parts []rio.AssemblyPart, expected filefixture.Fixture) {
	Convey("Assembly should not blow up", FailureContinues, func() {
		var assembly rio.Assembly
		So(func() {
			assembly = assemblerFn("./assembled", parts)
		}, ShouldNotPanic)
		Reset(func() {
			if assembly != nil {
				// conditional only because we may have continued moving after an error earlier.
				assembly.Teardown()
			}
		})

		Convey("Filesystem should scan as the expected union", func() {
			scan := filefixture.Scan("./assembled")
			So(scan.Describe(filefixture.CompareDefaults), ShouldEqual, expected.Describe(filefixture.CompareDefaults))
		})
	})
}

func CheckAssemblerRespectsReadonly(assemblerFn rio.Assembler) {
	Convey("Writing to a readonly placement should return EROFS",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				assembly := assemblerFn("./assembled", []rio.AssemblyPart{
					{TargetPath: "/", SourcePath: "./material/alpha", Writable: false},
				})
				defer assembly.Teardown()
				f, err := os.OpenFile("./assembled/newfile", os.O_CREATE, 0644)
				defer f.Close()
				So(err, ShouldNotBeNil)
				So(err, ShouldHaveSameTypeAs, &os.PathError{})
				So(err.(*os.PathError).Err, ShouldEqual, syscall.EROFS)
			}),
		),
	)
}

func CheckAssemblerIsolatesSource(assemblerFn rio.Assembler) {
	Convey("Writing to a placement should not alter the source",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				filefixture.Alpha.Create("./material/alpha")
				assembly := assemblerFn("./assembled", []rio.AssemblyPart{
					{TargetPath: "/", SourcePath: "./material/alpha", Writable: true},
				})
				defer assembly.Teardown()
				f, err := os.OpenFile("./assembled/newfile", os.O_CREATE, 0644)
				defer f.Close()
				So(err, ShouldBeNil)
				scan := filefixture.Scan("./material/alpha")
				So(scan.Describe(filefixture.CompareDefaults), ShouldEqual, filefixture.Alpha.Describe(filefixture.CompareDefaults))
			}),
		),
	)
}

func CheckAssemblerBareMount(assemblerFn rio.Assembler) {
	Convey("Bare mounts continue to see changes to the source",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				// make fixture
				filefixture.Alpha.Create("./material/alpha")
				// assemble
				assembly := assemblerFn("./assembled", []rio.AssemblyPart{
					{TargetPath: "/", SourcePath: "./material/alpha", Writable: false, BareMount: true},
				})
				defer assembly.Teardown()
				// modify on the outside
				f, err := os.OpenFile("./material/alpha/moar", os.O_CREATE, 0644)
				defer f.Close()
				So(err, ShouldBeNil)
				// the outside should see it (obviously! just a sanity check)
				So("./material/alpha/moar", testutil.ShouldBeFile, os.FileMode(0644))
				// the inside should see it
				So("./assembled/moar", testutil.ShouldBeFile, os.FileMode(0644))
			}),
		),
	)
	Convey("Writable bare mounts propagate changes to the source",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				// make fixture
				filefixture.Alpha.Create("./material/alpha")
				// assemble
				assembly := assemblerFn("./assembled", []rio.AssemblyPart{
					{TargetPath: "/", SourcePath: "./material/alpha", Writable: true, BareMount: true},
				})
				defer assembly.Teardown()
				// modify on the inside
				f, err := os.OpenFile("./assembled/moar", os.O_CREATE, 0644)
				defer f.Close()
				So(err, ShouldBeNil)
				// the inside should see it (obviously! just a sanity check)
				So("./material/alpha/moar", testutil.ShouldBeFile, os.FileMode(0644))
				// the outside should see it
				So("./assembled/moar", testutil.ShouldBeFile, os.FileMode(0644))
			}),
		),
	)
	Convey("Readonly bare mounts reject writes",
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				// make fixture
				filefixture.Alpha.Create("./material/alpha")
				// assemble
				assembly := assemblerFn("./assembled", []rio.AssemblyPart{
					{TargetPath: "/", SourcePath: "./material/alpha", Writable: false, BareMount: true},
				})
				defer assembly.Teardown()
				// modify on the inside should instantly error
				f, err := os.OpenFile("./assembled/moar", os.O_CREATE, 0644)
				defer f.Close()
				So(err, ShouldNotBeNil)
			}),
		),
	)
}
