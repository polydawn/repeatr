package chroot

import (
	"context"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/rio/stitch"
)

type Executor struct {
	assemblerTool *stitch.Assembler // Contains: unpackTool, caching cfg, and placer tools.
}

var _ repeatr.RunFunc = Executor{}.Run

func (cfg Executor) Run(
	ctx context.Context,
	formula api.Formula,
	input repeatr.InputControl,
	monitor repeatr.Monitor,
) (*api.RunRecord, error) {
	// Make work dirs.
	// TODO

	// Shell out to assembler.
	// TODO

	// Invoke containment and run!
	// TODO
	// DESIGN: this really only needs the `frm.Action` and the work dir...

	// Pack outputs.
	// TODO

	// Done!
	return nil, nil
}
