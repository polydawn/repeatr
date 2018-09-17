package tests

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	. "go.polydawn.net/repeatr/testutil"
)

func shouldRun(t *testing.T, runTool repeatr.RunFunc, frm api.Formula, frmCtx repeatr.FormulaContext) (api.FormulaRunRecord, string) {
	rr, txt, err := run(t, runTool, frm, baseFormulaCtx)
	AssertNoError(t, err)
	return *rr, txt
}
func run(t *testing.T, runTool repeatr.RunFunc, frm api.Formula, frmCtx repeatr.FormulaContext) (*api.FormulaRunRecord, string, error) {
	bm := bufferingMonitor{}
	rr, err := runTool(context.Background(), frm, baseFormulaCtx, repeatr.InputControl{}, bm.monitor())
	close(bm.Ch)
	bm.await()
	return rr, bm.Txt.String(), err
}

type bufferingMonitor struct {
	Ch  chan repeatr.Event
	Wg  sync.WaitGroup
	Txt bytes.Buffer
	Err error
}

func (bm *bufferingMonitor) monitor() repeatr.Monitor {
	*bm = bufferingMonitor{
		Ch: make(chan repeatr.Event),
	}
	bm.Wg.Add(1)
	go func() {
		defer bm.Wg.Done()
		for msg := range bm.Ch {
			bm.Err = dictateOutput(msg, &bm.Txt)
		}
	}()
	return repeatr.Monitor{Chan: bm.Ch}
}

func (bm *bufferingMonitor) await() error {
	bm.Wg.Wait()
	return bm.Err
}

func dictateOutput(evt repeatr.Event, into io.Writer) error {
	switch evt2 := evt.(type) {
	case repeatr.Event_Output:
		_, err := into.Write([]byte(evt2.Msg))
		return err
	}
	return nil
}
