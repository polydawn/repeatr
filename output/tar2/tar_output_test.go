package tar2

import (
	"testing"

	"polydawn.net/repeatr/output/tests"
)

func Test(t *testing.T) {
	tests.CheckScanWithoutMutation(t, "tar", New)
}
