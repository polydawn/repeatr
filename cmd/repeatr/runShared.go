package main

import (
	"context"
	"os"

	. "github.com/polydawn/go-errcat"
	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/repeatr/executor/impl/chroot"
	"go.polydawn.net/rio/client"
	"go.polydawn.net/rio/fs"
)

func run(
	ctx context.Context,
	executorName string,
	formulaPath string,
	inputControl repeatr.InputControl,
	monitor repeatr.Monitor,
) (rr *api.RunRecord, err error) {
	// Load and parse formula.
	f, err := os.Open(formulaPath)
	if err != nil {
		return nil, Errorf(repeatr.ErrUsage, "error opening formula file: %s", err)
	}
	var formulaUnion api.FormulaUnion
	if err := json.NewUnmarshallerAtlased(f, api.RepeatrAtlas).Unmarshal(&formulaUnion); err != nil {
		return nil, Errorf(repeatr.ErrUsage, "formula file does not parse: %s", err)
	}

	// Pack and unpack tools are always the Rio exec client.
	var (
		unpackTool rio.UnpackFunc = rioexecclient.UnpackFunc
		packTool   rio.PackFunc   = rioexecclient.PackFunc
	)

	// Demux executor implementation from name.
	var executor repeatr.RunFunc
	switch executorName {
	case "chroot":
		executor, err = chroot.NewExecutor(
			fs.MustAbsolutePath("/var/lib/timeless/repeatr/executor/chroot/"),
			unpackTool, packTool,
		)
		if err != nil {
			return nil, err
		}
	}

	// Invoke executor engine.
	return executor(
		ctx,
		formulaUnion.Formula,
		*formulaUnion.Context,
		inputControl,
		monitor,
	)
}
