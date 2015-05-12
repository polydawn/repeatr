package dir

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/tests"
)

func TestCoreCompliance(t *testing.T) {
	Convey("Spec Compliance: Dir Transmat", t, func() {
		tests.CheckScanWithoutMutation(integrity.TransmatKind("dir"), New)
		tests.CheckScanProducesConsistentHash(integrity.TransmatKind("dir"), New)
		tests.CheckScanProducesDistinctHashes(integrity.TransmatKind("dir"), New)
	})
}
