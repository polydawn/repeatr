package tar2

import (
	"testing"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output/tests"
)

func Test(t *testing.T) {
	tests.CheckScanWithoutMutation(t, New(def.Output{
		Type:     "tar",
		Location: "./data",
		URI:      "./output.tar",
	}))
}
