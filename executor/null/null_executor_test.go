package null

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/testutil"
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
					Warehouses: []string{"file://./fixture/beta"},
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

			result1 := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c}).Wait()
			result2 := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c}).Wait()
			So(result1.Outputs["out"].Hash, ShouldEqual, result2.Outputs["out"].Hash)
			// and changing a significant conjecture field should cause changes
			formula.Action = def.Action{Cwd: "/change"}
			result3 := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c}).Wait()
			So(result2.Outputs["out"].Hash, ShouldNotEqual, result3.Outputs["out"].Hash)
		})
	})
}
