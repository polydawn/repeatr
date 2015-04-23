package dir

import (
	"testing"

	"polydawn.net/repeatr/output/tests"
)

func TestCoreCompliance(t *testing.T) {
	tests.CheckScanWithoutMutation(t, "dir", New, "./output.dump")
	tests.CheckScanProducesConsistentHash(t, "dir", New, "./output.dump")
	tests.CheckScanProducesDistinctHashes(t, "dir", New, "./output.dump")
}
