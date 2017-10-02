package tests

import (
	"bytes"
	"context"
	"testing"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	. "go.polydawn.net/repeatr/testutil"
)

func shouldRun(t *testing.T, runTool repeatr.RunFunc, frm api.Formula, frmCtx api.FormulaContext) (api.RunRecord, string) {
	bm := bufferingMonitor{}.init()
	rr, err := runTool(context.Background(), frm, baseFormulaCtx, repeatr.InputControl{}, bm.monitor())
	WantNoError(t, err)
	return *rr, bm.Txt.String()
}

type bufferingMonitor struct {
	Txt bytes.Buffer
	Ch  chan repeatr.Event
	Err error
}

func (bm bufferingMonitor) init() *bufferingMonitor {
	bm = bufferingMonitor{
		Ch: make(chan repeatr.Event),
	}
	go func() { // leaks.  fuck the police.
		for {
			bm.Err = repeatr.CopyOut(<-bm.Ch, &bm.Txt)
		}
	}()
	return &bm
}
func (bm *bufferingMonitor) monitor() repeatr.Monitor {
	return repeatr.Monitor{Chan: bm.Ch}
}
