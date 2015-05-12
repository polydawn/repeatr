package tar

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/tests"
)

func TestCoreCompliance(t *testing.T) {
	Convey("Spec Compliance: Tar Transmat", t, func() {
		// scanning
		tests.CheckScanWithoutMutation(integrity.TransmatKind("tar"), New)
		tests.CheckScanProducesConsistentHash(integrity.TransmatKind("tar"), New)
		tests.CheckScanProducesDistinctHashes(integrity.TransmatKind("tar"), New)
		// round-trip
		tests.CheckRoundTrip(integrity.TransmatKind("tar"), New, "./bounce")
	})
}
