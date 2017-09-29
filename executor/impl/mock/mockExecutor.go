package mock

import (
	"context"
	"crypto/sha512"
	"strings"

	. "github.com/polydawn/go-errcat"
	"github.com/polydawn/refmt/misc"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/repeatr/executor/mixins"
)

type Executor struct {
}

var _ repeatr.RunFunc = Executor{}.Run

func (cfg Executor) Run(
	ctx context.Context,
	formula api.Formula,
	input repeatr.InputControl,
	monitor repeatr.Monitor,
) (*api.RunRecord, error) {
	// Only accept "mock" input and output specifications.
	//  Since this executor doesn't do any *real* executing, we certainly
	//  don't want to let it be used improperly accidentically.
	for _, inputWare := range formula.Inputs {
		if !strings.HasPrefix(string(inputWare.Type), "mock") {
			return nil, Errorf(repeatr.ErrUsage, "the mock executor can only run with mock inputs!")
		}
	}
	for _, outputSpec := range formula.Outputs {
		if !strings.HasPrefix(string(outputSpec.PackType), "mock") {
			return nil, Errorf(repeatr.ErrUsage, "the mock executor can only run with mock outputs!")
		}
	}

	// Start filling out record keeping!
	//  Includes picking a random guid for the job, which we use in all temp files.
	rr := &api.RunRecord{}
	mixins.InitRunRecord(rr, formula)

	// Fabricate outputs.
	//  The demo executor just *makes stuff up*, mostly deterministically
	//  based on the setup hash.
	for outputName, outputSpec := range formula.Outputs {
		hasher := sha512.New384()
		hasher.Write([]byte(rr.FormulaID))
		hasher.Write([]byte(outputName))
		rr.Results[outputName] = api.WareID{
			outputSpec.PackType,
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
