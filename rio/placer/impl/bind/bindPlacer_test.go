// +build linux

package bind

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/lib/testutil"
	"go.polydawn.net/repeatr/rio/placer"
	"go.polydawn.net/repeatr/rio/placer/tests"
)

func TestBindPlacerCompliance(t *testing.T) {
	assemblerFn := placer.NewAssembler(BindPlacer)
	Convey("Bind placers make data appear into place", t,
		testutil.Requires(
			testutil.RequiresMounts,
			func() {
				tests.CheckAssemblerGetsDataIntoPlace(assemblerFn)
			},
		),
	)
	Convey("Bind placers support readonly placement", t,
		testutil.Requires(
			testutil.RequiresMounts,
			func() {
				tests.CheckAssemblerRespectsReadonly(assemblerFn)
			},
		),
	)
	// Not Supported: CheckAssemblerIsolatesSource // (use AufsPlacer for that)
	// Not Supported: CheckAssemblerBareMount // (pointless, that's the only thing this one does)
}
