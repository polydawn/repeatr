package filefixture

import (
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/lib/fs"
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

	testutil.Convey_IfHaveRoot("Symlink breakouts should be refuted", t, FailureContinues, testutil.WithTmpdir(func() {
		// this is a sketchy, unsandboxed test.
		// I hope you don't have anything in /tmp/dangerzone, and/or that you're running the entire suite in a vm.
		os.RemoveAll("/tmp/dangerzone")
		Convey("With a relative basepath", func() {
			So(func() { Breakout.Create(".") }, testutil.ShouldPanicWith, fs.BreakoutError)
			_, err := os.Stat("/tmp/dangerzone/laaaaanaaa")
			So(err, ShouldNotBeNil) // if nil err, oh my god, it exists
		})
		Convey("With an absolute basepath", func() {
			pwd, err := os.Getwd()
			So(err, ShouldBeNil)
			So(func() { Breakout.Create(pwd) }, testutil.ShouldPanicWith, fs.BreakoutError)
			_, err = os.Stat("/tmp/dangerzone/laaaaanaaa")
			So(err, ShouldNotBeNil) // if nil err, oh my god, it exists
		})
	}))
}
