package main

import (
	"context"
	"io"

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

	// Invoke executor engine.
	rr, err := executor(
		ctx,
		api.Formula{}, // TODO load and parse
		repeatr.InputControl{},
		repeatr.Monitor{}, // TODO rig up IO proxy
	)
	if err != nil {
		return err // TODO probably print rr anyway
	}
	_ = rr

	return nil
}
