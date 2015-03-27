package filefixture

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	// uncomment for an example output
	//	Convey("Describe fixture Beta", t, func() {
	//		Println() // goconvey seems to do alignment rong in cli out of box :I
	//		Println(Beta.Describe(CompareAll))
	//	})

	testutil.Convey_IfHaveRoot("All fixtures should be able to apply their content to an empty dir", t, func() {
		for _, fixture := range All {
			Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
				fixture.Create(".")
				So(true, ShouldBeTrue) // reaching here is success
			}))
		}
	})

	testutil.Convey_IfHaveRoot("Applying a fixture and rescanning it should produce identical descriptions", t, func() {
		for _, fixture := range All {
			Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
				fixture.Create(".")
				reheat := Scan(".")
				So(reheat.Describe(CompareDefaults), ShouldEqual, fixture.Describe(CompareDefaults))
			}))
		}
	})
}
