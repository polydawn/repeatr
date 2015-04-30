/*
	Tests for use in each input implementation.
	Import and invoke in each implementation's package to get coverage
	within that package.
*/
package tests

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/output"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

type InputFactory func(def.Input) input.Input
type OutputFactory func(def.Output) output.Output

/*
	Checks round-trip hash consistency for a pair of input and output systems.

	- Creates a fixture filesystem
	- Scans it with the output system
	- Places it in a new filesystem with the input system and the scanned hash
	- Checks the new filesystem matches the original
*/
func CheckRoundTrip(t *testing.T, kind string, newOutput OutputFactory, newInput InputFactory) {
	Convey("Scanning and replacing a filesystem should agree on hash and content", t,
		testutil.Requires(testutil.RequiresRoot, func() {
			for _, fixture := range filefixture.All {
				Convey(fmt.Sprintf("- Fixture %q", fixture.Name), FailureContinues, testutil.WithTmpdir(func() {
					// setup fixture
					fixture.Create("./fixture")

					// scan with output
					scanner := newOutput((def.Output{
						Type: kind,
						URI:  "./output.dump",
					}))
					report := <-scanner.Apply("./fixture")
					So(report.Err, ShouldBeNil)

					// place with input (along the way, requires hash match)
					input := newInput((def.Input{
						Type: kind,
						Hash: report.Output.Hash,
						URI:  "./output.dump",
					}))
					err := <-input.Apply("./unpack")
					So(err, ShouldBeNil)

					// check filesystem to match original fixture
					// (do this check even if the input raised a hash mismatch, because it can help show why)
					rescan := filefixture.Scan("./unpack")
					comparisonLevel := filefixture.CompareDefaults &^ filefixture.CompareSubsecond
					So(rescan.Describe(comparisonLevel), ShouldEqual, fixture.Describe(comparisonLevel))
				}))
			}
		}),
	)
}
