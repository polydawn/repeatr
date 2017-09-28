package chroot

import (
	"context"

	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/repeatr/executor/mixins"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fs/osfs"
	"go.polydawn.net/rio/stitch"
)

type Executor struct {
	workspaceFs   fs.FS             // A working dir per execution will be made in here.
	assemblerTool *stitch.Assembler // Contains: unpackTool, caching cfg, and placer tools.
}

var _ repeatr.RunFunc = Executor{}.Run

func (cfg Executor) Run(
	ctx context.Context,
	formula api.Formula,
	input repeatr.InputControl,
	monitor repeatr.Monitor,
) (*api.RunRecord, error) {
	// Start filling out record keeping!
	//  Includes picking a random guid for the job, which we use in all temp files.
	rr := &api.RunRecord{}
	mixins.InitRunRecord(rr, formula)

	// Make work dirs.
	jobPath := fs.MustRelPath(rr.Guid)
	chrootPath := jobPath.Join(fs.MustRelPath("chroot"))
	if err := cfg.workspaceFs.Mkdir(jobPath, 0700); err != nil {
		return nil, Recategorize(err, repeatr.ErrLocalCacheProblem)
	}
	if err := cfg.workspaceFs.Mkdir(chrootPath, 0755); err != nil {
		return nil, Recategorize(err, repeatr.ErrLocalCacheProblem)
	}
	chrootFs := osfs.New(cfg.workspaceFs.BasePath().Join(chrootPath))

	// Shell out to assembler.
	unpackSpecs := stitch.FormulaToUnpackTree(formula, api.Filter_NoMutation)
	cleanupFunc, err := cfg.assemblerTool.Run(ctx, chrootFs, unpackSpecs)
	if err != nil {
		return nil, repeatr.ReboxRioError(err)
	}
	defer func() {
		if err := cleanupFunc(); err != nil {
			// TODO log it
		}
	}()

	// Invoke containment and run!
	// TODO
	// DESIGN: this really only needs the `frm.Action` and the work dir...

	// Pack outputs.
	// TODO

	// Done!
	return nil, nil
}
