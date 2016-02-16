package tar

import (
	"fmt"
	"os"
	"testing"

	"github.com/polydawn/gosh"
	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

func TestTarOutputCompat(t *testing.T) {
	Convey("Output should produce a tar recognizable to gnu tar", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func(c C) {
				for _, fixture := range filefixture.All {
					Convey(fmt.Sprintf("- Fixture %q", fixture.Name), func() {
						// create fixture
						fixture.Create("./data")

						// scan it
						transmat := New("./workdir")
						transmat.Scan(
							Kind,
							"./data",
							[]integrity.SiloURI{
								integrity.SiloURI("file://output.tar"),
							},
							testutil.TestLogger(c),
						)

						// sanity check that there's a file.
						So("./output.tar", testutil.ShouldBeFile, os.FileMode(0644))

						// now exec tar, and check that it doesn't barf outright.
						// this is not well isolated from the host; consider improving that a todo.
						os.Mkdir("./untar", 0755)
						tarProc := gosh.Gosh(
							"tar",
							"-xf", "./output.tar",
							"-C", "./untar",
							gosh.NullIO,
						).RunAndReport()
						So(tarProc.GetExitCode(), ShouldEqual, 0)

						// should look roughly the same again even bounced through
						// some third-party tar implementation, one would hope.
						rescan := filefixture.Scan("./untar")
						comparisonLevel := filefixture.CompareDefaults &^ filefixture.CompareSubsecond
						So(rescan.Describe(comparisonLevel), ShouldEqual, fixture.Describe(comparisonLevel))
					})
				}
			}),
		),
	)
}
