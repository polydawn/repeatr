package tests

import (
	"fmt"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

func CheckScanWithoutMutation(kind integrity.TransmatKind, transmatFabFn integrity.TransmatFactory) {
	Convey("SPEC: Scanning a filesystem shouldn't change it", testutil.Requires(
		testutil.RequiresRoot,
		testutil.WithTmpdir(func() {
			for _, fixture := range filefixture.All {
				transmat := transmatFabFn("./workdir")
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), func() {
					// set up fixture
					fixture.Create("./data")
					// scan it with the transmat
					silos := []integrity.SiloURI{"./dump"} // TODO this should be empty/nil for non-uploads, but we haven't taught implementations about that yet
					transmat.Scan(kind, "./data", silos)
					// rescan it with the test system
					rescan := filefixture.Scan("./data")
					// should be unchanged
					So(rescan.Describe(filefixture.CompareDefaults), ShouldEqual, fixture.Describe(filefixture.CompareDefaults))
				})
			}
		}),
	))
}
