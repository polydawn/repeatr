package demo

import (
	"context"
	"os"
	"time"

	. "go.polydawn.net/repeatr/lib/errcat"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/timeless-api"
	"go.polydawn.net/timeless-api/repeatr"
)

type Executor struct {
}

var _ repeatr.RunFunc = Executor{}.Run

func (cfg Executor) Run(
	ctx context.Context,
	formula *api.Formula,
	defaultWarehouses []api.WarehouseAddr, // default input warehouses
	outputWarehouses map[api.AbsPath][]api.WarehouseAddr, // output warehouses
	inputWarehouses map[api.AbsPath][]api.WarehouseAddr, // input override warehouses
	stream chan<- *repeatr.Event,
) (*api.RunRecord, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, Errorf(repeatr.ErrExecutor, "%s", err)
	}
	return &api.RunRecord{
		UID:       guid.New(),
		Time:      time.Now().Unix(),
		FormulaID: formula.SetupHash(),
		Results:   make(map[api.AbsPath]api.WareID),
		ExitCode:  0,
		Hostname:  hostname,
		Metadata:  make(map[string]string),
	}, nil
}
