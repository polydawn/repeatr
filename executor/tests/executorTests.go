package tests

import (
	"context"
	"testing"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	. "go.polydawn.net/repeatr/testutil"
)

var (
	// Base formula full of sensible defaults and ready to run:
	baseFormula = api.Formula{
		Inputs: map[api.AbsPath]api.WareID{
			"/": api.WareID{"tar", "6q7G4hWr283FpTa5Lf8heVqw9t97b5VoMU6AGszuBYAz9EzQdeHVFAou7c4W9vFcQ6"},
		},
		Action: api.FormulaAction{
			Exec: []string{"/bin/echo", "hello world"},
		},
	}
	baseFormulaCtx = api.FormulaContext{
		FetchUrls: map[api.AbsPath][]api.WarehouseAddr{
			"/": []api.WarehouseAddr{
				"file://../../../fixtures/busybash.tgz",
			},
		},
	}
)

func CheckHelloWorldTxt(t *testing.T, runTool repeatr.RunFunc) {
	t.Run("hello-world formula should work", func(t *testing.T) {
		frm := baseFormula.Clone()

		bm := bufferingMonitor{}.init()
		rr, err := runTool(context.Background(), frm, baseFormulaCtx, repeatr.InputControl{}, bm.monitor())
		WantNoError(t, err)

		t.Run("exit code should be success", func(t *testing.T) {
			WantEqual(t, rr.ExitCode, 0)
		})
		t.Run("txt should be echo'd string", func(t *testing.T) {
			WantEqual(t, bm.Txt.String(), "hello world\n")
		})
	})
}

func CheckRoundtripRootfs(t *testing.T, runTool repeatr.RunFunc) {
	t.Run("output unmodified root fileset should work", func(t *testing.T) {
		frm := baseFormula.Clone()
		frm.Outputs = map[api.AbsPath]api.OutputSpec{
			"/": {PackType: "tar", Filters: api.Filter_NoMutation},
		}

		bm := bufferingMonitor{}.init()
		rr, err := runTool(context.Background(), frm, baseFormulaCtx, repeatr.InputControl{}, bm.monitor())
		WantNoError(t, err)

		t.Run("output ware from '/' should be familiar!", func(t *testing.T) {
			WantEqual(t, map[api.AbsPath]api.WareID{
				"/": baseFormula.Inputs["/"],
			}, rr.Results)
		})
	})
}
