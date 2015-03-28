/*
	Tests for use in each output implementation.
	Import and invoke in each implementation's package to get coverage
	within that package.
*/
package tests

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

func CheckScanWithoutMutation(t *testing.T, subject output.Output) {
	testutil.Convey_IfHaveRoot("Applying the output to a filesystem shouldn't change it", t, func() {
		for _, fixture := range filefixture.All {
			Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
				fixture.Create("./data")
				So(<-subject.Apply("./data"), ShouldBeNil)
				rescan := filefixture.Scan("./data")
				So(rescan.Describe(filefixture.CompareDefaults), ShouldResemble, fixture.Describe(filefixture.CompareDefaults))
			}))
		}
	})
}
