package main

import (
	"bytes"
	"fmt"
	"io"
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

func loadBasting(bastingPath string) (*api.Basting, error) {
	f, err := os.Open(bastingPath)
	if err != nil {
		return nil, Errorf(repeatr.ErrUsage, "error opening basting file: %s", err)
	}
	var slot api.Basting
	if err := json.NewUnmarshallerAtlased(f, api.HitchAtlas).Unmarshal(&slot); err != nil {
		return nil, Errorf(repeatr.ErrUsage, "basting file does not parse: %s", err)
	}
	return &slot, nil
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

func printRunRecord(stdout, stderr io.Writer, rr *api.RunRecord) {
	if rr == nil {
		return
	}
	// Buffer rather than go direct to stdout so we can flush with linebreaks at the same time.
	//  This makes output slightly more readable (otherwise a stderr write can get stuck
	//  dangling after the runrecord...).
	var buf bytes.Buffer
	if err := json.NewMarshallerAtlased(&buf, jsonPrettyOptions, api.RepeatrAtlas).Marshal(rr); err != nil {
		fmt.Fprintf(stderr, "%s\n", err)
	}
	buf.Write([]byte{'\n'})
	stdout.Write(buf.Bytes())
}
