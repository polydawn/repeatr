package null

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/lib/testutil"
)

func Test(t *testing.T) {
	Convey("Nil Executor mocking utilities", t, func(c C) {
		execEng := &Executor{}
		execEng.Configure("null_workspace")
		formula := def.Formula{
			Inputs: def.InputGroup{
				"part2": &def.Input{
					Type:       "dir",
					Hash:       "asegdrh",
					Warehouses: def.WarehouseCoords{"file://./fixture/beta"},
					MountPath:  "/data/test",
				},
			},
			Outputs: def.OutputGroup{
				"out": &def.Output{
					Type:       "mock",
					Conjecture: true,
				},
			},
		}

		Convey("Deterministic mode should have consistent results", func() {
			execEng.Mode = Deterministic

			result1 := execEng.Start(formula, executor.JobID(guid.New()), nil, testutil.TestLogger(c)).Wait()
			result2 := execEng.Start(formula, executor.JobID(guid.New()), nil, testutil.TestLogger(c)).Wait()
			So(result1.Outputs["out"].Hash, ShouldEqual, result2.Outputs["out"].Hash)
			// and changing a significant conjecture field should cause changes
			formula.Action = def.Action{Cwd: "/change"}
			result3 := execEng.Start(formula, executor.JobID(guid.New()), nil, testutil.TestLogger(c)).Wait()
			So(result2.Outputs["out"].Hash, ShouldNotEqual, result3.Outputs["out"].Hash)
		})
	})
}
