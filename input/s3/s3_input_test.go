package s3

import (
	"testing"

	"github.com/rlmcpherson/s3gof3r"
	"polydawn.net/repeatr/input/tests"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/output/s3"
)

func TestCoreCompliance(t *testing.T) {
	if _, err := s3gof3r.EnvKeys(); err != nil {
		t.Skipf("skipping s3 output tests; no s3 credentials loaded (err: %s)", err)
	}

	// group all effects of this test run under one "dir" for human reader sanity and cleanup in extremis.
	testRunGuid := guid.New()

	tests.CheckRoundTrip(t, "s3", s3.New, New, "s3://repeatr-test/test-"+testRunGuid+"/rt/obj.tar")
	tests.CheckRoundTrip(t, "s3", s3.New, New, "s3+splay://repeatr-test/test-"+testRunGuid+"/rt-splay/heap/")
}
