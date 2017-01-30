package copy

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/rio/placer"
	"go.polydawn.net/repeatr/rio/placer/tests"
)

func TestCopyingPlacerCompliance(t *testing.T) {
	assemblerFn := placer.NewAssembler(CopyingPlacer)
	Convey("Copying placers make data appear into place", t, func() {
		tests.CheckAssemblerGetsDataIntoPlace(assemblerFn)
	})
	// Not Supported: CheckAssemblerRespectsReadonly
	Convey("Copying placers support source isolation", t, func() {
		tests.CheckAssemblerIsolatesSource(assemblerFn)
	})
	// Not Supported: CheckAssemblerBareMount // (can't do live changes with cp)
}
