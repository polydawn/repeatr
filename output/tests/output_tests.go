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
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

type OutputFactory func(def.Output) output.Output

func CheckScanWithoutMutation(t *testing.T, kind string, newOutput OutputFactory) {
	testutil.Convey_IfHaveRoot("Applying the output to a filesystem shouldn't change it", t, func() {
		for _, fixture := range filefixture.All {
			Convey(fmt.Sprintf("- Fixture %q", fixture.Name), testutil.WithTmpdir(func() {
				subject := newOutput((def.Output{
					Type:     kind,
					Location: "./data",
					URI:      "./output.dump", // assumes your output supports local file for throwaway :/
				}))
				fixture.Create("./data")
				So(<-subject.Apply("./data"), ShouldBeNil)
				rescan := filefixture.Scan("./data")
				So(rescan.Describe(filefixture.CompareDefaults), ShouldResemble, fixture.Describe(filefixture.CompareDefaults))
			}))
		}
	})
}
