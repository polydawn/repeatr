package tar

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/tests"
	"polydawn.net/repeatr/testutil"
)

func TestCoreCompliance(t *testing.T) {
	Convey("Spec Compliance: Tar Transmat", t, testutil.WithTmpdir(func() {
		// scanning
		tests.CheckScanWithoutMutation(integrity.TransmatKind("tar"), New)
		tests.CheckScanProducesConsistentHash(integrity.TransmatKind("tar"), New)
		tests.CheckScanProducesDistinctHashes(integrity.TransmatKind("tar"), New)
		// round-trip
		tests.CheckRoundTrip(integrity.TransmatKind("tar"), New, "file://bounce", "file literal", "relative")
		// FIXME REQUIRES TEST REFACTOR // cwd, _ := os.Getwd() // WRONG CWD.
		// tests.CheckRoundTrip(integrity.TransmatKind("tar"), New, "file://"+filepath.Join(cwd, "bounce"), "file literal", "absolute")
		// round-trip using content-addressible "warehouse"
		// FIXME REQUIRES TEST REFACTOR // os.Mkdir("bounce", 0755) // WRONG CWD.
		tests.CheckRoundTrip(integrity.TransmatKind("tar"), New, "file+ca://bounce", "content-addressible")
	}))
}
