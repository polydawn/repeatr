package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
)

func Run(
	ctx context.Context,
	executorName string,
	formulaPath string,
	stdout, stderr io.Writer,
) (err error) {
	// Prepare monitor and IO forwarding.
	evtChan := make(chan repeatr.Event)
	monitor := repeatr.Monitor{evtChan}
	monitorWg := sync.WaitGroup{}
	monitorWg.Add(1)
	go func() {
		defer monitorWg.Done()
		for {
			select {
			case evt, ok := <-evtChan:
				if !ok {
					return
				}
				switch {
				case evt.Log != nil:
					fmt.Fprintf(stderr, "log: lvl=%s msg=%s\n", evt.Log.Level, evt.Log.Msg)
				case evt.Output != nil:
					repeatr.CopyOut(evt, stderr)
				case evt.Result != nil:
					// pass
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	inputControl := repeatr.InputControl{}

	// Call helper for all the bits that are in common with twerk mode
	//  (load formula, demux stuff, actually launch).
	rr, err := run(
		ctx,
		executorName,
		formulaPath,
		inputControl,
		monitor,
	)
	monitorWg.Wait()

	// If a runrecord was returned always try to print it, even if we have
	//  an error and thus it may be incomplete.
	if rr != nil {
		if err := json.NewMarshallerAtlased(stdout, api.RepeatrAtlas).Marshal(rr); err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
		stdout.Write([]byte{'\n'})
	}
	// Return the executor error.
	return err
}
