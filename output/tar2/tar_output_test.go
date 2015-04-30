package tar2

import (
	"fmt"
	"os"
	"testing"

	"github.com/polydawn/gosh"
	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output/tests"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

func TestCoreCompliance(t *testing.T) {
	tests.CheckScanWithoutMutation(t, "tar", New)
	tests.CheckScanProducesConsistentHash(t, "tar", New)
	tests.CheckScanProducesDistinctHashes(t, "tar", New)
}

func TestTarCompat(t *testing.T) {
	testutil.Convey_IfHaveRoot("Applying the output to a filesystem should produce a tar file", t, func() {
		for _, fixture := range filefixture.All {
			Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
				subject := New(def.Output{
					Type: "tar",
					URI:  "./output.tar",
				})
				fixture.Create("./data")
				report := <-subject.Apply("./data")
				// sanity check that it worked, and that there's a file.
				So(report.Err, ShouldBeNil)
				So("./output.tar", testutil.ShouldBeFile, os.FileMode(0))
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
				// boy, that's entertaining though: gnu tar does all the same stuff,
				// except it doesn't honor our nanosecond timings.
				comparisonLevel := filefixture.CompareDefaults &^ filefixture.CompareSubsecond
				So(rescan.Describe(comparisonLevel), ShouldEqual, fixture.Describe(comparisonLevel))
			}))
		}
	})
}
