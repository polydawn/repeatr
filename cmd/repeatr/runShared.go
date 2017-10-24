package main

import (
	"os"

	. "github.com/polydawn/go-errcat"
	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/repeatr/executor/impl/chroot"
	"go.polydawn.net/repeatr/executor/impl/runc"
	"go.polydawn.net/rio/client"
	"go.polydawn.net/rio/fs"
)

func loadFormula(formulaPath string) (*api.Formula, *api.FormulaContext, error) {
	f, err := os.Open(formulaPath)
	if err != nil {
		return nil, nil, Errorf(repeatr.ErrUsage, "error opening formula file: %s", err)
	}
	var slot api.FormulaUnion
	if err := json.NewUnmarshallerAtlased(f, api.RepeatrAtlas).Unmarshal(&slot); err != nil {
		return nil, nil, Errorf(repeatr.ErrUsage, "formula file does not parse: %s", err)
	}
	return &slot.Formula, slot.Context, nil
}

func demuxExecutor(executorName string) (repeatr.RunFunc, error) {
	// Pack and unpack tools are always the Rio exec client.
	var (
		unpackTool rio.UnpackFunc = rioexecclient.UnpackFunc
		packTool   rio.PackFunc   = rioexecclient.PackFunc
	)

	// Demux executor implementation from name.
	switch executorName {
	case "chroot":
		return chroot.NewExecutor(
			fs.MustAbsolutePath("/var/lib/timeless/repeatr/executor/chroot/"),
			unpackTool, packTool,
		)
	case "runc":
		return runc.NewExecutor(
			fs.MustAbsolutePath("/var/lib/timeless/repeatr/executor/runc/"),
			unpackTool, packTool,
		)
	default:
		return nil, Errorf(repeatr.ErrUsage, "not a known executor: %q", executorName)
	}
}
