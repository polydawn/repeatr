package mock

import (
	"context"
	"crypto/sha512"
	"os"
	"strings"
	"time"

	"github.com/polydawn/refmt/misc"

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
	// Only accept "mock" input and output specifications.
	//  Since this executor doesn't do any *real* executing, we certainly
	//  don't want to let it be used improperly accidentically.
	for _, inputWare := range formula.Inputs {
		if !strings.HasPrefix(inputWare.Type, "mock") {
			return nil, Errorf(repeatr.ErrUsage, "the mock executor can only run with mock inputs!")
		}
	}
	for _, outputSpec := range formula.Outputs {
		if !strings.HasPrefix(outputSpec, "mock") {
			return nil, Errorf(repeatr.ErrUsage, "the mock executor can only run with mock outputs!")
		}
	}

	// Gather host environment data.
	//  This will be reported in the runrecord as some advisory/logging metadata.
	hostname, err := os.Hostname()
	if err != nil {
		return nil, Errorf(repeatr.ErrExecutor, "%s", err)
	}

	// Start filling out RunRecord.
	//  Even in case of error, we will return this much.
	setupHash := formula.SetupHash()
	rr := &api.RunRecord{
		UID:       guid.New(),
		Time:      time.Now().Unix(),
		FormulaID: setupHash,
		Results:   make(map[api.AbsPath]api.WareID),
		ExitCode:  -1,
		Hostname:  hostname,
	}

	// Fabricate outputs.
	//  The demo executor just *makes stuff up*, mostly deterministically
	//  based on the setup hash.
	for outputName, packType := range formula.Outputs {
		hasher := sha512.New384()
		hasher.Write([]byte(setupHash))
		hasher.Write([]byte(outputName))
		hasher.Write([]byte(packType))
		rr.Results[outputName] = api.WareID{
			packType,
			misc.Base58Encode(hasher.Sum(nil)),
		}
	}
	// FUTURE: this executor could read in some more clues from the
	//  formula.Action perhaps, for switching behavior on things like
	//  exit code and determinism?
	rr.ExitCode = 0

	// Done!
	return rr, nil
}
