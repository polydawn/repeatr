package mixins

import (
	"context"

	. "github.com/warpfork/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/repeatr/executor/cradle"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/stitch"
)

func WithFilesystem(
	ctx context.Context,
	chrootFs fs.FS, // Unpack everything here.
	assemblerTool *stitch.Assembler, // Using this tool.
	packTool rio.PackFunc, // And this tool.
	formula api.Formula, // Following these instructions.
	formulaCtx repeatr.FormulaContext, // Fetching and saving from here.
	mon repeatr.Monitor, // Logging to this.
	fn func(fs.FS) error, // Then call this while it's set up.
) (results map[api.AbsPath]api.WareID, err error) {
	defer RequireErrorHasCategory(&err, repeatr.ErrorCategory(""))

	// Shell out to assembler.
	unpackSpecs := unpackSpecsForFormula(formula, formulaCtx, api.FilesetUnpackFilter_Lossless)
	wgRioLogs := ForwardRioUnpackLogs(ctx, mon, unpackSpecs)
	cleanupFunc, err := assemblerTool.Run(ctx, chrootFs, unpackSpecs, cradle.DirpropsForUserinfo(*formula.Action.Userinfo))
	wgRioLogs.Wait()
	if err != nil {
		return nil, repeatr.ReboxRioError(err)
	}
	defer CleanupFuncWithLogging(cleanupFunc, mon)()

	// Last bit of filesystem brushup: run cradle fs mutations.
	if err := cradle.TidyFilesystem(formula, chrootFs); err != nil {
		return nil, err
	}

	// Do the thing!
	if err := fn(chrootFs); err != nil {
		return nil, err
	}

	// Pack outputs.
	packSpecs := packSpecsForFormula(formula, formulaCtx, api.FilesetPackFilter_Flatten)
	results, err = stitch.PackMulti(ctx, packTool, chrootFs, packSpecs)
	return results, repeatr.ReboxRioError(err)
}

/*
	Reduce a formula to a slice of []stitch.UnpackSpec, ready to be used
	invoking stitch.Assembler.Run().

	Typically the filters arg will be either `api.FilesetUnpackFilter_Lossless`
	or `api.FilesetUnpackFilter_LowPriv`, depending if you're using repeatr or
	rio respectively, though other values are of course valid.

	Whether the action, outputs, or saveUrls are set is irrelevant;
	they will be ignored completely.
*/
func unpackSpecsForFormula(frm api.Formula, frmCtx repeatr.FormulaContext, filters api.FilesetUnpackFilter) (parts []stitch.UnpackSpec) {
	for path, wareID := range frm.Inputs {
		warehouses, _ := frmCtx.FetchUrls[path]
		parts = append(parts, stitch.UnpackSpec{
			Path:       fs.MustAbsolutePath(string(path)),
			WareID:     wareID,
			Filters:    filters,
			Warehouses: warehouses,
		})
	}
	return
}

/*
	Reduce a formula to a slice of []stitch.PackSpec, ready to be used
	invoking stitch.PackMulti().

	The filters given will be applied to all *unset* fields in the filters
	already given by the formula outputs; set fields are not changed.
	Typically the filters arg will be `api.Filter_DefaultFlatten`
	(both `rio pack` and repeatr outputs default to this),
	though other values are of course valid.

	Whether the action, inputs, or fetchUrls are set is irrelevant;
	they will be ignored completely.
*/
func packSpecsForFormula(frm api.Formula, frmCtx repeatr.FormulaContext, filters api.FilesetPackFilter) (parts []stitch.PackSpec) {
	for path, output := range frm.Outputs {
		warehouse, _ := frmCtx.SaveUrls[path]
		parts = append(parts, stitch.PackSpec{
			Path:      fs.MustAbsolutePath(string(path)),
			PackType:  output.PackType,
			Filter:    output.Filter.Apply(filters),
			Warehouse: warehouse,
		})
	}
	return
}
