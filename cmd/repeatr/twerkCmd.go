package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	"go.polydawn.net/go-timeless-api/repeatr"
)

func Twerk(
	ctx context.Context,
	executorName string,
	formulaPath string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (err error) {
	// Load formula and build executor.
	executor, err := demuxExecutor(executorName)
	if err != nil {
		return err
	}
	formula, formulaContext, err := loadFormula(formulaPath)
	if err != nil {
		return err
	}

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

	// Run!  (And wait for output forwarding worker to finish.)
	rr, err := executor(
		ctx,
		*formula,
		*formulaContext,
		inputControl,
		monitor,
	)
	monitorWg.Wait()

	// If a runrecord was returned always try to print it, even if we have
	//  an error and thus it may be incomplete.
	printRunRecord(stdout, stderr, rr)
	// Return the executor error.
	return err
}
