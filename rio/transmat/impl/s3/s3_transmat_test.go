package s3

import (
	"testing"

	"github.com/rlmcpherson/s3gof3r"
	. "github.com/smartystreets/goconvey/convey"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/lib/testutil"
	"go.polydawn.net/repeatr/rio/tests"
)

func TestCoreCompliance(t *testing.T) {
	if _, err := s3gof3r.EnvKeys(); err != nil {
		t.Skipf("skipping s3 output tests; no s3 credentials loaded (err: %s)", err)
	}

	// group all effects of this test run under one "dir" for human reader sanity and cleanup in extremis.
	testRunGuid := guid.New()

	Convey("Spec Compliance: S3 Transmat", t, testutil.WithTmpdir(func() {
		// scanning
		tests.CheckScanWithoutMutation(Kind, New)
		tests.CheckScanProducesConsistentHash(Kind, New)
		tests.CheckScanProducesDistinctHashes(Kind, New)
		tests.CheckScanEmptyIsCalm(Kind, New)
		tests.CheckScanWithFilters(Kind, New)
		// round-trip
		tests.CheckRoundTrip(Kind, New, "s3://repeatr-test/test-"+testRunGuid+"/bounce/obj.tar", "literal path")
		tests.CheckRoundTrip(Kind, New, "s3+ca://repeatr-test/test-"+testRunGuid+"/bounce-ca/heap/", "content addressible path")
	}))
}
