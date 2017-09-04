package core

import (
	"context"

	"go.polydawn.net/repeatr/executor"
	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
)

type Runner struct {
	Executor executor.Interface
}

var _ repeatr.RunFunc = Runner{}.Run

/*
	Things that this method will do:

		- Everything fun.  That means:
			- Conjure Wares and Assemble Filesets.
			- Spawn containers.  Run things.
			- Teardown filesystem, save new Wares.
			- Tell you about it.

	Things that should already have been done:

		- One metric ton of args parsing: the CLI (or whatever other caller)
		   should have finished this by now.  The formula is *done*.
		- The selection of executors.  We just want to shell out to one.
		- The selection of transmats.  We just want to shell out to some.
*/
func (cfg Runner) Run(
	ctx context.Context,
	formula *api.Formula,
	defaultWarehouses []api.WarehouseAddr, // default input warehouses
	outputWarehouses map[api.AbsPath][]api.WarehouseAddr, // output warehouses
	inputWarehouses map[api.AbsPath][]api.WarehouseAddr, // input override warehouses
	stream chan<- *repeatr.Event,
) (*api.RunRecord, error) {
	return nil, nil
}
