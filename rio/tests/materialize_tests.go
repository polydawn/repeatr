package tests

import (
	"fmt"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/lib/testutil"
	"go.polydawn.net/repeatr/lib/testutil/filefixture"
	"go.polydawn.net/repeatr/rio"
)

/*
	Checks round-trip hash consistency for the input and output halves of a transmat system.

	- Creates a fixture filesystem
	- Scans it with the output system
	- Places it in a new filesystem with the input system and the scanned hash
	- Checks the new filesystem matches the original
*/
func CheckRoundTrip(kind rio.TransmatKind, transmatFabFn rio.TransmatFactory, bounceURI string, addtnlDesc ...string) {
	Convey("SPEC: Round-trip scanning and remaking a filesystem should agree on hash and content"+testutil.AdditionalDescription(addtnlDesc...), testutil.Requires(
		testutil.RequiresRoot,
		func(c C) {
			transmat := transmatFabFn("./workdir")
			log := testutil.TestLogger(c)

			for _, fixture := range filefixture.All {
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), FailureContinues, func() {
					uris := []rio.SiloURI{rio.SiloURI(bounceURI)}
					// setup fixture
					fixture.Create("./fixture")
					// scan it with the transmat
					dataHash := transmat.Scan(kind, "./fixture", uris, log)
					// materialize what we just scanned (along the way, requires hash match)
					arena := transmat.Materialize(kind, dataHash, uris, log, rio.AcceptHashMismatch)
					// assert hash match
					// (normally survival would attest this, but we used the `AcceptHashMismatch` to supress panics in the name of letting the test see more after failures.)
					So(arena.Hash(), ShouldEqual, dataHash)
					// check filesystem to match original fixture
					// (do this check even if the input raised a hash mismatch, because it can help show why)
					rescan := filefixture.Scan(arena.Path())
					comparisonLevel := filefixture.CompareDefaults &^ filefixture.CompareSubsecond
					So(rescan.Describe(comparisonLevel), ShouldEqual, fixture.Describe(comparisonLevel))
				})
			}
		},
	))
}
