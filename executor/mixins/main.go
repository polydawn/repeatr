package mixins

import (
	"context"

	. "github.com/polydawn/go-errcat"

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
	formulaCtx api.FormulaContext, // Fetching and saving from here.
	mon repeatr.Monitor, // Logging to this.
	fn func(fs.FS) error, // Then call this while it's set up.
) (results map[api.AbsPath]api.WareID, err error) {
	defer RequireErrorHasCategory(&err, repeatr.ErrorCategory(""))

	// Shell out to assembler.
	unpackSpecs := stitch.FormulaToUnpackSpecs(formula, formulaCtx, api.Filter_NoMutation)
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
	packSpecs := stitch.FormulaToPackSpecs(formula, formulaCtx, api.Filter_DefaultFlatten)
	results, err = stitch.PackMulti(ctx, packTool, chrootFs, packSpecs)
	return results, repeatr.ReboxRioError(err)
}
