/*
	Bind an `executor.Executor` into service and
	expose it as an `api/act.RunObserver`.
*/
package runner

import (
	"fmt"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/act"
	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/lib/guid"
)

var (
	_ act.RunProvider = &Runner{}
	_ act.RunObserver = &Runner{}
)

func New(cfg Config) *Runner {
	return &Runner{cfg: cfg}
}

type Runner struct {
	cfg Config
	state
}

type Config struct {
	Executor executor.Executor
}

type state struct {
	runID def.RunID // picked when `StartRun` called.
}

// TODO both of the following methods are supposed to return immediately.
// We need another goroutine to be provided for actual power.

func (r *Runner) StartRun(*def.Formula) def.RunID {
	if r.runID != "" {
		panic(meep.Meep(
			&meep.ErrProgrammer{},
			meep.Cause(fmt.Errorf("invalid use of Runner; can only start once")),
		))
	}
	// Assign an ID.
	r.runID = def.RunID(guid.New())
	// Return immediately.  We don't actually start until output is connected.
	return r.runID
}

/*
	Implements `act.RunObserver`.

	It is absolutely necessary to call this.  Runner will not begin
	executing your formula until the event output stream has been hooked up.
	It is also necessary for the caller to service the event stream;
	if the event channel is full and cannot accept additional events,
	regular operations (e.g. write to stderr) for the program
	inside your formula may block.

	Only good for the one RunID this runner is in charge of; all others
	will result in a panic of `*act.ErrRunIDNotFound` as you'd expect.

	Sequence IDs will be ignored.  You'll get the entire stream from the
	beginning.

	This method is does not allow multiple calls -- it cannot be used for
	resume/seek -- but it is reliable (it is not subject to network failure
	modes).
	For network and resume/seek use, apply buffering layers above this.
*/
func (r *Runner) FollowEvents(
	which def.RunID,
	stream chan<- *def.Event,
	_ def.EventSeqID,
) {
	// Verify arg validity.
	if which != r.runID {
		panic(meep.Meep(&act.ErrRunIDNotFound{RunID: which}))
	}
	// TODO submit channel into actor control loop
	// Return immediately.
}
