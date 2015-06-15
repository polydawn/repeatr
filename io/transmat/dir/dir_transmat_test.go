package dir

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io/tests"
)

func TestCoreCompliance(t *testing.T) {
	Convey("Spec Compliance: Dir Transmat", t, func() {
		// scanning
		tests.CheckScanWithoutMutation(Kind, New)
		tests.CheckScanProducesConsistentHash(Kind, New)
		tests.CheckScanProducesDistinctHashes(Kind, New)
		// round-trip
		tests.CheckRoundTrip(Kind, New, "./bounce")
	})
}
