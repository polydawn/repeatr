package file

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.polydawn.net/repeatr/lib/testutil"
)

func TestCoreCompliance(t *testing.T) {
	Convey("Spec Compliance: File Transmat", t, testutil.WithTmpdir(func() {
		// Mercy.  Without scan implemented, none of our test standards work.
	}))
}
