package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	. "github.com/warpfork/go-errcat"

	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/repeatr/fmt"
)

func Twerk(
	ctx context.Context,
	executorName string,
	formulaPath string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (err error) {
	defer RequireErrorHasCategory(&err, repeatr.ErrorCategory(""))

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
				switch evt2 := evt.(type) {
				case repeatr.Event_Log:
					fmt.Fprintf(stderr, "log: lvl=%s msg=%s\n", evt2.Level, evt2.Msg)
				case repeatr.Event_Output:
					stderr.Write([]byte(evt2.Msg))
				case repeatr.Event_Result:
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
	close(monitor.Chan)
	monitorWg.Wait()

	// If a runrecord was returned always try to print it, even if we have
	//  an error and thus it may be incomplete.
	repeatrfmt.NewAnsiPrinter(stdout, stderr).PrintResult(repeatr.Event_Result{rr, repeatr.ToError(err)})
	// Return the executor error.
	return err
}
