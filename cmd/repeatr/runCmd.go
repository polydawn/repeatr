package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	"go.polydawn.net/go-timeless-api/repeatr"
)

func Run(
	ctx context.Context,
	executorName string,
	formulaPath string,
	stdout, stderr io.Writer,
	memoDir string,
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

	// Consider possibility of memoization.
	//  If a memo dir is set and it contains a relevant record, we just echo it.
	rr, err := loadMemo(formula.SetupHash(), memoDir)
	if err != nil {
		return err
	}
	if rr != nil {
		fmt.Fprintf(stderr, "log: lvl=%s msg=%s\n", repeatr.LogInfo, "memoized runRecord found for formula setupHash; eliding run")
		printRunRecord(stdout, stderr, rr)
		return nil
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

	// Run!  (And wait for output forwarding worker to finish.)
	rr, err = executor(
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
