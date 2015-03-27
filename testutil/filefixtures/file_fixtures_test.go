package filefixture

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	testutil.Convey_IfHaveRoot("Checking that fixtures can apply", t, func() {
		for _, fixture := range All {
			Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
				fixture.Create(".")
				So(true, ShouldBeTrue) // reaching here is success
			}))
		}
	})
}
