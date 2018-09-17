package main

import (
	"os"

	"github.com/polydawn/refmt/json"
	"github.com/polydawn/refmt/obj/atlas"
	. "github.com/warpfork/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/go-timeless-api/rio/client/exec"
	"go.polydawn.net/repeatr/executor/impl/chroot"
	"go.polydawn.net/repeatr/executor/impl/gvisor"
	"go.polydawn.net/repeatr/executor/impl/runc"
	"go.polydawn.net/rio/fs"
)

type (
	// formulaPlus is the concatenation of a formula and its context, and is
	// useful to serialize both {the thing to do} and {what you need to do it}
	// for sending to a repeatr process as one complete message.
	formulaPlus struct {
		Formula api.Formula
		Context repeatr.FormulaContext
	}
)

var (
	formulaPlus_AtlasEntry = atlas.BuildEntry(formulaPlus{}).StructMap().Autogenerate().Complete()

	atl_formulaPlus = atlas.MustBuild(
		formulaPlus_AtlasEntry,
		api.Formula_AtlasEntry,
		api.FilesetPackFilter_AtlasEntry,
		api.FormulaAction_AtlasEntry,
		api.FormulaUserinfo_AtlasEntry,
		api.FormulaOutputSpec_AtlasEntry,
		api.WareID_AtlasEntry,
		repeatr.FormulaContext_AtlasEntry,
	)
)

func loadFormula(formulaPath string) (*api.Formula, *repeatr.FormulaContext, error) {
	f, err := os.Open(formulaPath)
	if err != nil {
		return nil, nil, Errorf(repeatr.ErrUsage, "error opening formula file: %s", err)
	}
	var slot formulaPlus
	if err := json.NewUnmarshallerAtlased(f, atl_formulaPlus).Unmarshal(&slot); err != nil {
		return nil, nil, Errorf(repeatr.ErrUsage, "formula file does not parse: %s", err)
	}
	return &slot.Formula, &slot.Context, nil
}

func demuxExecutor(executorName string) (repeatr.RunFunc, error) {
	// Pack and unpack tools are always the Rio exec client.
	var (
		unpackTool rio.UnpackFunc = rioclient.UnpackFunc
		packTool   rio.PackFunc   = rioclient.PackFunc
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
	case "gvisor":
		return gvisor.NewExecutor(
			fs.MustAbsolutePath("/var/lib/timeless/repeatr/executor/gvisor/"),
			unpackTool, packTool,
		)
	default:
		return nil, Errorf(repeatr.ErrUsage, "not a known executor: %q", executorName)
	}
}
