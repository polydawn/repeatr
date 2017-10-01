package mock

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
)

func Test(t *testing.T) {
	var e repeatr.RunFunc = Executor{}.Run

	Convey("Mock Executor sanity tests", t, func() {
		formula := api.Formula{
			Inputs: map[api.AbsPath]api.WareID{
				"/data/test": api.WareID{"mocktar", "weofijqweoi"},
			},
			Action: api.FormulaAction{
				Exec: []string{"thing"},
			},
			Outputs: map[api.AbsPath]api.OutputSpec{
				"/out": api.OutputSpec{PackType: "mocktar"},
			},
		}
		formulaCtx := api.FormulaContext{}

		Convey("Should produce results", func() {
			rr1, err := e(
				context.Background(),
				formula, formulaCtx,
				repeatr.InputControl{}, repeatr.Monitor{},
			)
			So(err, ShouldBeNil)
			So(rr1.Results, ShouldHaveLength, 1)

			Convey("Should produce *consistent* results", func() {
				rr2, err := e(
					context.Background(),
					formula, formulaCtx,
					repeatr.InputControl{}, repeatr.Monitor{},
				)
				So(err, ShouldBeNil)
				So(rr1.Results, ShouldResemble, rr2.Results)
			})

			Convey("Changing the formula should produce different results", func() {
				formula.Action = api.FormulaAction{Exec: []string{"differentthing"}}
				rr2, err := e(
					context.Background(),
					formula, formulaCtx,
					repeatr.InputControl{}, repeatr.Monitor{},
				)
				So(err, ShouldBeNil)
				So(rr1.Results, ShouldNotResemble, rr2.Results)
			})
		})
	})
}
