package s3

import (
	"testing"

	"github.com/rlmcpherson/s3gof3r"
	"polydawn.net/repeatr/output/tests"
)

// Note: most of the interesting tests are over in the paired inputs package;
//  those demonstrate that we any of our interactions with S3 actually worked
//   (beyond the level of "it didn't tell us it blew up" covered here).

func TestCoreCompliance(t *testing.T) {
	// FIXME: do a cleanup pass on these shared test things...
	//   - they shouldn't be doing requirements (like ifHaveRoot) internally; that's caller's choice.
	//   - they shouldn't be taking `t`, because that limits composability.
	//   - looks like we're gonna need to be able to supply the URI to bounce to, as well.

	if _, err := s3gof3r.EnvKeys(); err != nil {
		t.Skipf("skipping s3 output tests; no s3 credentials loaded (err: %s)", err)
	}

	tests.CheckScanWithoutMutation(t, "s3", New)
	tests.CheckScanProducesConsistentHash(t, "s3", New)
	tests.CheckScanProducesDistinctHashes(t, "s3", New)
}
