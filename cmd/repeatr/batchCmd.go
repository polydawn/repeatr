package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	. "github.com/polydawn/go-errcat"
	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/repeatr/batch"
	"go.polydawn.net/rio/fs"
)

func BatchCmd(
	ctx context.Context,
	executorName string,
	bastingPath string,
	stdout, stderr io.Writer,
	memoDir *fs.AbsolutePath,
) (err error) {
	defer RequireErrorHasCategory(&err, repeatr.ErrorCategory(""))

	printer := &ansi{stdout: stdout, stderr: stderr} // todo: switch

	// Load basting & compute evaluation order.
	basting, err := loadBasting(bastingPath)
	if err != nil {
		return err
	}
	printer.printLog(repeatr.Event_Log{
		Time:  time.Now(),
		Level: repeatr.LogInfo,
		Msg:   "calculating evaluation dependency order...",
		Detail: [][2]string{
			{"graphSize", fmt.Sprintf("%d", len(basting.Steps))},
		},
	})
	stepOrder, err := batch.OrderSteps(*basting)
	if err != nil {
		return Errorf(repeatr.ErrUsage, "structurally invalid basting: %s", err)
	}
	printer.printLog(repeatr.Event_Log{
		Time:  time.Now(),
		Level: repeatr.LogInfo,
		Msg:   "calculated evaluation order!",
		Detail: [][2]string{
			{"graphSize", fmt.Sprintf("%d", len(basting.Steps))},
			{"order", fmt.Sprintf("%s", stepOrder)},
		},
	})

	// Run stuff!  In order.
	//  This is placeholder implementation quality.  We should be
	//  exec'ing each of these, and thereby also be ready to parallelize.
	runRecords := map[string]api.RunRecord{}
	for stepNum, stepName := range stepOrder {
		formula, imports, formulaCtx := basting.Steps[stepName].Formula,
			basting.Steps[stepName].Imports, basting.Contexts[stepName]
		for path, imp := range imports {
			if imp.CatalogName == "wire" {
				formula.Inputs[path] = runRecords[string(imp.ReleaseName)].Results[api.AbsPath(imp.ItemName)]
				printer.printLog(repeatr.Event_Log{
					Time:  time.Now(),
					Level: repeatr.LogInfo,
					Msg:   "wire import resolved",
					Detail: [][2]string{
						{"stepName", stepName},
						{"stepNum", fmt.Sprintf("%d/%d", stepNum+1, len(stepOrder))},
						{"path", string(path)},
						{"import", imp.String()},
						{"resolved", formula.Inputs[path].String()},
					},
				})
				break
			}
		}
		rr, err := Run(ctx, executorName, formula, formulaCtx, printer, memoDir)
		if err != nil {
			printer.printLog(repeatr.Event_Log{
				Time:  time.Now(),
				Level: repeatr.LogError,
				Msg:   "executor reports error",
				Detail: [][2]string{
					{"err", err.Error()},
				},
			})
			return err
		}
		runRecords[stepName] = *rr
	}

	// Now that we've finished all steps, we can print all the final export wares:
	//  (FUTURE: this format should be something almost ready to pipe into a `hitch commit` command!)
	exports := map[api.ItemName]api.WareID{}
	for itemName, wire := range basting.Exports {
		exports[itemName] = runRecords[string(wire.ReleaseName)].Results[api.AbsPath(wire.ItemName)]
	}
	printBatchResults(stdout, stderr, exports)

	return nil
}

func printBatchResults(stdout, stderr io.Writer, exports map[api.ItemName]api.WareID) {
	// Buffer rather than go direct to stdout so we can flush with linebreaks at the same time.
	//  This makes output slightly more readable (otherwise a stderr write can get stuck
	//  dangling after the runrecord...).
	var buf bytes.Buffer
	if err := json.NewMarshallerAtlased(&buf, jsonPrettyOptions, api.HitchAtlas).Marshal(exports); err != nil {
		fmt.Fprintf(stderr, "%s\n", err)
	}
	buf.Write([]byte{'\n'})
	stdout.Write(buf.Bytes())
}
