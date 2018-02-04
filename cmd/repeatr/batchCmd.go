package main

import (
	"context"
	"fmt"
	"io"

	. "github.com/polydawn/go-errcat"

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

	// Load basting & compute evaluation order.
	basting, err := loadBasting(bastingPath)
	if err != nil {
		return err
	}
	stepOrder, err := batch.OrderSteps(*basting)
	if err != nil {
		return Errorf(repeatr.ErrUsage, "structurally invalid basting: %s", err)
	}
	fmt.Printf("orden: %s\n", stepOrder)

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
				fmt.Fprintf(stderr, "log: lvl=%s msg=%s %v\n", repeatr.LogInfo, "wire import resolved", [][2]string{
					{"stepName", stepName},
					{"stepNum", fmt.Sprintf("%d/%d", stepNum+1, len(stepOrder))},
					{"path", string(path)},
					{"import", imp.String()},
					{"resolved", formula.Inputs[path].String()},
				})
				break
			}
		}
		rr, err := Run(ctx, executorName, formula, formulaCtx, stdout, stderr, memoDir)
		if err != nil {
			fmt.Fprintf(stderr, "log: lvl=%s msg=%s %v\n", repeatr.LogError, "executor reports error", [][2]string{
				{"err", err.Error()},
			})
			return err
		}
		runRecords[stepName] = *rr
	}

	return nil
}
