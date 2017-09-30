package main

import (
	"context"
	"fmt"
	"io"
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

func Run(
	ctx context.Context,
	executorName string,
	formulaPath string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (err error) {
	// Load and parse formula.
	f, err := os.Open(formulaPath)
	if err != nil {
		return Errorf(repeatr.ErrUsage, "error opening formula file: %s", err)
	}
	var formula api.Formula
	if err := json.NewUnmarshallerAtlased(f, api.RepeatrAtlas).Unmarshal(&formula); err != nil {
		return Errorf(repeatr.ErrUsage, "formula does not parse: %s", err)
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
			return err
		}
	}

	// Prepare monitor and IO forwarding.
	evtChan := make(chan repeatr.Event)
	go func() { // leaks.  fuck the police.
		for {
			repeatr.CopyOut(<-evtChan, stderr)
		}
	}()

	// Invoke executor engine.
	rr, err := executor(
		ctx,
		formula,
		repeatr.InputControl{},
		repeatr.Monitor{evtChan},
	)
	// Always attempt to emit the runrecord json, even if we have an error
	//  and it may be incomplete.
	if err := json.NewMarshallerAtlased(stdout, api.RepeatrAtlas).Marshal(rr); err != nil {
		fmt.Fprintf(stderr, "%s", err)
	}
	// Return the executor error.
	return err
}
