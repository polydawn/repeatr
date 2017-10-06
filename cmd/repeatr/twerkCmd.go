package main

import (
	"context"
	"fmt"
	"io"

	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
)

func Twerk(
	ctx context.Context,
	executorName string,
	formulaPath string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (err error) {
	// Prepare monitor and IO forwarding.
	evtChan := make(chan repeatr.Event)
	monitor := repeatr.Monitor{evtChan}
	go func() {
		for {
			repeatr.CopyOut(<-evtChan, stderr)
		}
	}()
	inputControl := repeatr.InputControl{}
	if stdin != nil {
		inputChan := make(chan string)
		inputControl.Chan = inputChan
		go func() {
			buf := [1024]byte{}
			for {
				n, err := stdin.Read(buf[:])
				if err != nil {
					if err == io.EOF {
						close(inputChan)
						return
					}
					fmt.Fprintf(stderr, "%s\n", err)
					return
				}
				inputChan <- string(buf[0:n])
				// TODO Blocking.  If you want this to "DTRT" for an
				// interactive terminal, sending those IOCTLs is something
				// you must have done already.
			}
		}()
	}

	// Call helper for all the bits that are in common with twerk mode
	//  (load formula, demux stuff, actually launch).
	rr, err := run(
		ctx,
		executorName,
		formulaPath,
		inputControl,
		monitor,
	)

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
