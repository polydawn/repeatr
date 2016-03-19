package tests

import (
	"fmt"
	"os"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/filter"
	"polydawn.net/repeatr/lib/testutil"
	"polydawn.net/repeatr/lib/testutil/filefixture"
)

// TODO : surprisingly few of these tests cover actually saving content to a warehouse.
//  While that seems fine within the definitions of the word "scan", and we
//   do have coverage via the round-trip tests, we could do a much better job
//   of testing the commit-to-remote concern as an isolated unit... if we
//   had more APIs around warehouse state inspection.  Big task.  Tackle soon.

func CheckScanWithoutMutation(kind rio.TransmatKind, transmatFabFn rio.TransmatFactory) {
	Convey("SPEC: Scanning a filesystem shouldn't change it", testutil.Requires(
		testutil.RequiresRoot,
		func(c C) {
			for _, fixture := range filefixture.All {
				transmat := transmatFabFn("./workdir")
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), func() {
					// set up fixture
					fixture.Create("./data")
					// scan it with the transmat
					transmat.Scan(kind, "./data", nil, testutil.TestLogger(c))
					// rescan it with the test system
					rescan := filefixture.Scan("./data")
					// should be unchanged
					So(rescan.Describe(filefixture.CompareDefaults), ShouldEqual, fixture.Describe(filefixture.CompareDefaults))
				})
			}
		},
	))
}

func CheckScanProducesConsistentHash(kind rio.TransmatKind, transmatFabFn rio.TransmatFactory) {
	Convey("SPEC: Applying the output to a filesystem twice should produce the same hash", testutil.Requires(
		testutil.RequiresRoot,
		func(c C) {
			for _, fixture := range filefixture.All {
				transmat := transmatFabFn("./workdir")
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), func() {
					// set up fixture
					fixture.Create("./data")
					// scan it with the transmat
					commitID1 := transmat.Scan(kind, "./data", nil, testutil.TestLogger(c))
					// scan it with again
					commitID2 := transmat.Scan(kind, "./data", nil, testutil.TestLogger(c))
					// should be same output
					So(commitID2, ShouldEqual, commitID1)
				})
			}
		},
	))
}

func CheckScanProducesDistinctHashes(kind rio.TransmatKind, transmatFabFn rio.TransmatFactory) {
	Convey("SPEC: Applying the output to two different filesystems should produce different hashes", testutil.Requires(
		testutil.RequiresRoot,
		func(c C) {
			transmat := transmatFabFn("./workdir")
			// set up fixtures
			filefixture.Alpha.Create("./alpha")
			filefixture.Beta.Create("./beta")
			// scan both filesystems with the transmat
			commitID1 := transmat.Scan(kind, "./alpha", nil, testutil.TestLogger(c))
			commitID2 := transmat.Scan(kind, "./beta", nil, testutil.TestLogger(c))
			// should be distinct
			So(commitID2, ShouldNotEqual, commitID1)
		}),
	)
}

func CheckScanEmptyIsCalm(kind rio.TransmatKind, transmatFabFn rio.TransmatFactory) {
	Convey("SPEC: Scanning a nonexistent filesystem should return an empty commitID", func(c C) {
		transmat := transmatFabFn("./workdir")
		commitID := transmat.Scan(kind, "./does-not-exist", nil, testutil.TestLogger(c))
		So(commitID, ShouldEqual, "")
	})
}

func CheckScanWithFilters(kind rio.TransmatKind, transmatFabFn rio.TransmatFactory) {
	Convey("SPEC: Filesystems only differing by mtime should have same hash after mtime filter", testutil.Requires(
		testutil.RequiresRoot,
		func(c C) {
			transmat := transmatFabFn("./workdir")
			// set up fixtures
			filefixture.Alpha.Create("./alpha1")
			filefixture.Alpha.Create("./alpha2")
			// overwrite the time on one of them -- can be nonconstant value, even; that's sorta the point.
			So(os.Chtimes("./alpha2/a", time.Now(), time.Now()), ShouldBeNil)
			// set up a filter.  can set their times to anything, as long as its the same
			filt := filter.MtimeFilter{time.Unix(1000000, 9000)}
			// scan both filesystems with the transmat
			commitID1 := transmat.Scan(kind, "./alpha1", nil, testutil.TestLogger(c), rio.UseFilter(filt))
			commitID2 := transmat.Scan(kind, "./alpha2", nil, testutil.TestLogger(c), rio.UseFilter(filt))
			// should be same
			So(commitID2, ShouldEqual, commitID1)
		}),
	)

	Convey("SPEC: Filesystems only differing by uid/gid should have same hash after filter", testutil.Requires(
		testutil.RequiresRoot,
		func(c C) {
			transmat := transmatFabFn("./workdir")
			// set up fixtures
			filefixture.Alpha.Create("./alpha1")
			filefixture.Alpha.Create("./alpha2")
			// overwrite the time on one of them -- can be nonconstant value, even; that's sorta the point.
			So(os.Chown("./alpha2/a", 908234, 20954), ShouldBeNil)
			// set up a filter.  can set their times to anything, as long as its the same
			ufilt := filter.UidFilter{10401}
			gfilt := filter.GidFilter{10401}
			// scan both filesystems with the transmat
			commitID1 := transmat.Scan(kind, "./alpha1", nil, testutil.TestLogger(c), rio.UseFilter(ufilt), rio.UseFilter(gfilt))
			commitID2 := transmat.Scan(kind, "./alpha2", nil, testutil.TestLogger(c), rio.UseFilter(ufilt), rio.UseFilter(gfilt))
			// should be same
			So(commitID2, ShouldEqual, commitID1)
		}),
	)
}

/*
	Commits the same content, twice, serially.

	This does not expose race conditions (and indeed, is not intended to,
	since some storage systems -- almost any of the non-CA ones, really --
	are not in fact race-safe), but it does check for sanity
	around the basic operations of convergence, esp in CA storage.
*/
func CheckMultipleCommit(kind rio.TransmatKind, transmatFabFn rio.TransmatFactory, bounceURI string, addtnlDesc ...string) {
	Convey("SPEC: Committing the same content twice must be safe"+testutil.AdditionalDescription(addtnlDesc...), testutil.Requires(
		testutil.RequiresRoot,
		func(c C) {
			transmat := transmatFabFn("./workdir")
			// set up fixtures
			filefixture.Alpha.Create("./alpha")
			// scan twice with the transmat, and commit to warehouse
			uris := []rio.SiloURI{rio.SiloURI(bounceURI)}
			commitID1 := transmat.Scan(kind, "./alpha", uris, testutil.TestLogger(c))
			commitID2 := transmat.Scan(kind, "./alpha", uris, testutil.TestLogger(c))
			// survival is winning: the main test was actually that scan #2 didn't panic.
			// sanity check: both commits should have been same hash
			So(commitID2, ShouldEqual, commitID1)
		},
	))
}
