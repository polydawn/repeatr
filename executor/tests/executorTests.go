package tests

import (
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
		frm, frmCtx := baseFormula.Clone(), baseFormulaCtx
		rr, txt := shouldRun(t, runTool, frm, frmCtx)
		t.Run("exit code should be success", func(t *testing.T) {
			WantEqual(t, rr.ExitCode, 0)
		})
		t.Run("txt should be echo'd string", func(t *testing.T) {
			WantEqual(t, txt, "hello world\n")
		})
	})
}

func CheckRoundtripRootfs(t *testing.T, runTool repeatr.RunFunc) {
	t.Run("output unmodified root fileset should work", func(t *testing.T) {
		frm, frmCtx := baseFormula.Clone(), baseFormulaCtx
		frm.Outputs = map[api.AbsPath]api.OutputSpec{
			"/": {PackType: "tar", Filters: api.Filter_NoMutation},
		}
		rr, _ := shouldRun(t, runTool, frm, frmCtx)
		t.Run("output ware from '/' should be familiar!", func(t *testing.T) {
			WantEqual(t, map[api.AbsPath]api.WareID{
				"/": baseFormula.Inputs["/"],
			}, rr.Results)
		})
	})
}

func CheckReportingExitCodes(t *testing.T, runTool repeatr.RunFunc) {
	t.Run("non-zero exits should report cleanly", func(t *testing.T) {
		frm, frmCtx := baseFormula.Clone(), baseFormulaCtx
		frm.Action = api.FormulaAction{
			Exec: []string{"/bin/bash", "-c", "exit 14"},
		}
		rr, _ := shouldRun(t, runTool, frm, frmCtx)
		WantEqual(t, rr.ExitCode, 14)
	})
}
