package gs

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/lib/testutil"
	"polydawn.net/repeatr/rio/tests"
)

func TestCoreCompliance(t *testing.T) {
	token, err := GetAccessToken()
	if err != nil {
		t.Skipf("skipping gs output tests; no gs credentials loaded (err: %s)", err)
	}
	if token == nil {
		t.Fatalf("No error, yet missing token (╯°□°）╯︵ ┻━┻")
	}
	// group all effects of this test run under one "dir" for human reader sanity and cleanup in extremis.
	testRunGuid := guid.New()

	Convey("Spec Compliance: GS Transmat", t, testutil.WithTmpdir(func() {
		// scanning
		tests.CheckScanWithoutMutation(Kind, New)
		tests.CheckScanProducesConsistentHash(Kind, New)
		tests.CheckScanProducesDistinctHashes(Kind, New)
		tests.CheckScanEmptyIsCalm(Kind, New)
		tests.CheckScanWithFilters(Kind, New)
		// round-trip
		tests.CheckRoundTrip(Kind, New, "gs://repeatr-test/test-"+testRunGuid+"/bounce/obj.tar", "literal path")
		tests.CheckRoundTrip(Kind, New, "gs+ca://repeatr-test/test-"+testRunGuid+"/bounce-ca/heap/", "content addressible path")
	}))
}
