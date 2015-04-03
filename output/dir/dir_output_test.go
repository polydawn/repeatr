package dir

import (
	"testing"

	"polydawn.net/repeatr/output/tests"
)

func TestCoreCompliance(t *testing.T) {
	tests.CheckScanWithoutMutation(t, "dir", New)
	tests.CheckScanProducesConsistentHash(t, "dir", New)
	tests.CheckScanProducesDistinctHashes(t, "dir", New)
}
