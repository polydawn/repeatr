/*
	Tests for use in each output implementation.
	Import and invoke in each implementation's package to get coverage
	within that package.
*/
package tests

import (
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

type OutputFactory func(def.Output) output.Output

func CheckScanWithoutMutation(t *testing.T, kind string, newOutput OutputFactory) {
	Convey("Applying the output to a filesystem shouldn't change it", t,
		testutil.Requires(testutil.RequiresRoot, func() {
			for _, fixture := range filefixture.All {
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
					subject := newOutput((def.Output{
						Type: kind,
						URI:  "./output.dump", // assumes your output supports local file for throwaway :/
					}))
					fixture.Create("./data")
					report := <-subject.Apply("./data")
					So(report.Err, ShouldBeNil)
					os.Remove("./output.dump")
					rescan := filefixture.Scan("./data")
					So(rescan.Describe(filefixture.CompareDefaults), ShouldResemble, fixture.Describe(filefixture.CompareDefaults))
				}))
			}
		}),
	)
}

func CheckScanProducesConsistentHash(t *testing.T, kind string, newOutput OutputFactory) {
	Convey("Applying the output to a filesystem twice should produce the same hash", t,
		testutil.Requires(testutil.RequiresRoot, func() {
			for _, fixture := range filefixture.All {
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
					fixture.Create("./data")
					scanner1 := newOutput((def.Output{
						Type: kind,
						URI:  "./output.dump",
					}))
					report1 := <-scanner1.Apply("./data")
					So(report1.Err, ShouldBeNil)
					os.RemoveAll("./output.dump")
					scanner2 := newOutput((def.Output{
						Type: kind,
						URI:  "./output.dump",
					}))
					report2 := <-scanner2.Apply("./data")
					So(report2.Err, ShouldBeNil)
					os.RemoveAll("./output.dump")
					So(report2.Output.Hash, ShouldEqual, report1.Output.Hash)
				}))
			}
		}),
	)
}

func CheckScanProducesDistinctHashes(t *testing.T, kind string, newOutput OutputFactory) {
	Convey("Applying the output to two different filesystems should produce different hashes", t,
		testutil.Requires(testutil.RequiresRoot, testutil.WithTmpdir(func() {
			filefixture.Alpha.Create("./alpha")
			filefixture.Alpha.Create("./beta")
			scanner1 := newOutput((def.Output{
				Type: kind,
				URI:  "./output.dump",
			}))
			report1 := <-scanner1.Apply("./alpha")
			So(report1.Err, ShouldBeNil)
			os.RemoveAll("./output.dump")
			scanner2 := newOutput((def.Output{
				Type: kind,
				URI:  "./output.dump",
			}))
			report2 := <-scanner2.Apply("./beta")
			So(report2.Err, ShouldBeNil)
			os.RemoveAll("./output.dump")
			So(report2.Output.Hash, ShouldEqual, report1.Output.Hash)
		})),
	)
}
