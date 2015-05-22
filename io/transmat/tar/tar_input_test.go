package tar

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/polydawn/gosh"
	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fspatch"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

const ubuntuTarballHash = "b6nXWuXamKB3TfjdzUSL82Gg1avuvTk0mWQP4wgegscZ_ZzG9GfHDwKXQ9BfCx6v"

func TestTarCompat(t *testing.T) {
	projPath, _ := os.Getwd()
	projPath = filepath.Dir(filepath.Dir(filepath.Dir(projPath)))

	Convey("Unpacking tars should match exec untar", t,
		testutil.Requires(testutil.RequiresRoot, testutil.WithTmpdir(func() {
			Convey("Given a fixture tarball containing ubuntu",
				testutil.Requires(testutil.RequiresLongRun, func() {
					transmat := New("./workdir/tar")

					// apply it; hope it doesn't blow up
					arena := transmat.Materialize(
						integrity.TransmatKind("tar"),
						integrity.CommitID(ubuntuTarballHash),
						[]integrity.SiloURI{
							integrity.SiloURI(filepath.Join(projPath, "assets/ubuntu.tar.gz")),
						},
					)
					defer arena.Teardown()

					// do a native untar; since we don't have an upfront fixture
					//  for this thing, we'll compare the two as filesystems.
					// this is not well isolated from the host; consider improving that a todo.
					os.Mkdir("./untar", 0755)
					tarProc := gosh.Gosh(
						"tar",
						"-xf", filepath.Join(projPath, "assets/ubuntu.tar.gz"),
						"-C", "./untar",
						gosh.NullIO,
					).RunAndReport()
					So(tarProc.GetExitCode(), ShouldEqual, 0)
					// native untar does not have an opinion about the base dir...
					// but our scans do, so, flatten that here
					So(fspatch.LUtimesNano("./untar", def.Epochwhen, def.Epochwhen), ShouldBeNil)

					// scan and compare
					scan1 := filefixture.Scan(arena.Path())
					scan2 := filefixture.Scan("./untar")
					// boy, that's entertaining though: gnu tar does all the same stuff,
					//  except it doesn't honor our nanosecond timings.
					// also exclude bodies because they're *big*.
					comparisonLevel := filefixture.CompareDefaults &^ filefixture.CompareSubsecond &^ filefixture.CompareBody
					So(scan1.Describe(comparisonLevel), ShouldEqual, scan2.Describe(comparisonLevel))
				}),
			)
		})),
	)
}
