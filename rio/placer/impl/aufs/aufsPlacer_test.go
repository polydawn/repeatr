// +build linux

package aufs

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/lib/testutil"
	"go.polydawn.net/repeatr/rio/placer"
	"go.polydawn.net/repeatr/rio/placer/tests"
)

func TestAufsPlacerCompliance(t *testing.T) {
	Convey("Aufs placers make data appear into place", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewAufsPlacer("./aufs-layers"))
				tests.CheckAssemblerGetsDataIntoPlace(assemblerFn)
			}),
		),
	)
	Convey("Aufs placers support readonly placement", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewAufsPlacer("./aufs-layers"))
				tests.CheckAssemblerRespectsReadonly(assemblerFn)
			}),
		),
	)
	Convey("Aufs placers support source isolation", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewAufsPlacer("./aufs-layers"))
				tests.CheckAssemblerIsolatesSource(assemblerFn)
			}),
		),
	)
	Convey("Aufs placers support bare mounts (non-isolation)", t,
		testutil.Requires(
			testutil.RequiresMounts,
			testutil.WithTmpdir(func() {
				assemblerFn := placer.NewAssembler(NewAufsPlacer("./aufs-layers"))
				tests.CheckAssemblerBareMount(assemblerFn)
			}),
		),
	)
}
