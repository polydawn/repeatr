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
					transmat.Scan(kind, "./data", nil)
					// rescan it with the test system
					rescan := filefixture.Scan("./data")
					// should be unchanged
					So(rescan.Describe(filefixture.CompareDefaults), ShouldEqual, fixture.Describe(filefixture.CompareDefaults))
				})
			}
		}),
	))
}

func CheckScanProducesConsistentHash(kind integrity.TransmatKind, transmatFabFn integrity.TransmatFactory) {
	Convey("SPEC: Applying the output to a filesystem twice should produce the same hash", testutil.Requires(
		testutil.RequiresRoot,
		testutil.WithTmpdir(func() {
			for _, fixture := range filefixture.All {
				transmat := transmatFabFn("./workdir")
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), func() {
					// set up fixture
					fixture.Create("./data")
					// scan it with the transmat
					commitID1 := transmat.Scan(kind, "./data", nil)
					// scan it with again
					commitID2 := transmat.Scan(kind, "./data", nil)
					// should be same output
					So(commitID2, ShouldEqual, commitID1)
				})
			}
		}),
	))
}

func CheckScanProducesDistinctHashes(kind integrity.TransmatKind, transmatFabFn integrity.TransmatFactory) {
	Convey("SPEC: Applying the output to two different filesystems should produce different hashes", testutil.Requires(
		testutil.RequiresRoot,
		testutil.WithTmpdir(func() {
			transmat := transmatFabFn("./workdir")
			// set up fixtures
			filefixture.Alpha.Create("./alpha")
			filefixture.Beta.Create("./beta")
			// scan both filesystems with the transmat
			commitID1 := transmat.Scan(kind, "./alpha", nil)
			commitID2 := transmat.Scan(kind, "./beta", nil)
			// should be distinct
			So(commitID2, ShouldNotEqual, commitID1)
		})),
	)
}
