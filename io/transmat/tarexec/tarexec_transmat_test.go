package tarexec

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/tests"
)

func TestCoreCompliance(t *testing.T) {
	Convey("Spec Compliance: TarExec Transmat", t, func() {
		// scanning
		tests.CheckScanWithoutMutation(integrity.TransmatKind("tar"), New)
		// WILL NOT PASS (no hashes!) -- tests.CheckScanProducesConsistentHash(integrity.TransmatKind("tar"), New)
		// WILL NOT PASS (no hashes!) -- tests.CheckScanProducesDistinctHashes(integrity.TransmatKind("tar"), New)
		// round-trip
		// WILL NOT PASS (no hashes!) -- tests.CheckRoundTrip(integrity.TransmatKind("tar"), New, "./bounce")
	})
}
