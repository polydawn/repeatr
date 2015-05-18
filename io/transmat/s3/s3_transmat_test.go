package s3

import (
	"testing"

	"github.com/rlmcpherson/s3gof3r"
	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/tests"
)

func TestCoreCompliance(t *testing.T) {
	if _, err := s3gof3r.EnvKeys(); err != nil {
		t.Skipf("skipping s3 output tests; no s3 credentials loaded (err: %s)", err)
	}

	Convey("Spec Compliance: S3 Transmat", t, func() {
		// scanning
		tests.CheckScanWithoutMutation(integrity.TransmatKind("s3"), New)
		tests.CheckScanProducesConsistentHash(integrity.TransmatKind("s3"), New)
		tests.CheckScanProducesDistinctHashes(integrity.TransmatKind("s3"), New)
		// round-trip
		tests.CheckRoundTrip(integrity.TransmatKind("s3"), New, "s3://repeatr-test/bounce")
	})
}
