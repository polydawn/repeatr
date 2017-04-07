// +build linux

package overlay

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/lib/testutil"
	"go.polydawn.net/repeatr/rio/placer"
	"go.polydawn.net/repeatr/rio/placer/tests"
)

func TestOverlayPlacerCompliance(t *testing.T) {
	Convey("Overlay placers make data appear into place", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewOverlayPlacer("./overlay-layers"))
				tests.CheckAssemblerGetsDataIntoPlace(assemblerFn)
			}),
		),
	)
	Convey("Overlay placers support readonly placement", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewOverlayPlacer("./overlay-layers"))
				tests.CheckAssemblerRespectsReadonly(assemblerFn)
			}),
		),
	)
	Convey("Overlay placers support source isolation", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewOverlayPlacer("./overlay-layers"))
				tests.CheckAssemblerIsolatesSource(assemblerFn)
			}),
		),
	)
	Convey("Overlay placers support bare mounts (non-isolation)", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewOverlayPlacer("./overlay-layers"))
				tests.CheckAssemblerBareMount(assemblerFn)
			}),
		),
	)
}
