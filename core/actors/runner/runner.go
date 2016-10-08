/*
	Bind an `executor.Executor` into service and
	expose it as an `api/act.RunObserver`.
*/
package runner

import (
	"fmt"
	"io"
	"sync"

	"go.polydawn.net/go-sup"
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
	return &Runner{
		cfg: cfg,
		state: state{
			rigging: make(chan chan<- *def.Event),
		},
	}
}

type Runner struct {
	cfg   Config
	mutex sync.Mutex
	state
}

type Config struct {
	Executor executor.Executor
	Stdin    io.Reader // hack for interactive mode.
}

type state struct {
	runID   def.RunID    // picked when `StartRun` called.
	frm     *def.Formula // set when `StartRun` called.
	rigging chan chan<- *def.Event
}

/*
	Main method for this actor; park a goroutine here to power
	execution.
*/
func (a *Runner) Run(supvr sup.Supervisor) {
	// Awaiting-launch phase.
	var stream chan<- *def.Event
	select {
	case stream = <-a.state.rigging:
	case <-supvr.QuitCh():
		return
	}

	// Set up logs, and launch executor.
	// (Executor doesn't know about go-sup yet; this may look different
	//  and have more obvious error flow paths when it's updated for that.)
	logSetup := evtStreamLogHandler{a.runID, stream}
	job := a.cfg.Executor.Start(
		*a.frm,
		executor.JobID(a.runID),
		a.cfg.Stdin,
		logSetup.NewLogger(),
	)

	// Service IO.
	// (Interruptability is poor around here; should drive supervisors farther.)
	// (Future work might involve removing the executor's buffer behavior
	//  entirely, which would also remove these goroutines for re-reading it.)
	jobOutputReader := job.Outputs().Reader(1, 2)
	journalWriter := &evtStreamJournalWriter{a.runID, stream}
	_, err := io.Copy(journalWriter, jobOutputReader)
	if err != nil {
		// This error path shouldn't be possible after we de-kink buffers.
		panic(err)
	}

	// Process final report.
	// Push log level events in addition to the runRecord
	rr := jobToRunRecord(job)
	stream <- &def.Event{
		RunID:     a.runID,
		RunRecord: rr,
	}
}

/*
	Request execution of a formula.
	Returns immediately with a RunID that will identify this run.
*/
func (r *Runner) StartRun(frm *def.Formula) def.RunID {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.runID != "" {
		panic(meep.Meep(
			&meep.ErrProgrammer{},
			meep.Cause(fmt.Errorf("invalid use of Runner; can only start once")),
		))
	}
	// Save formula.
	r.frm = frm.Clone() // paranoia clone.
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
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// Verify arg validity.
	if which != r.runID {
		panic(meep.Meep(&act.ErrRunIDNotFound{RunID: which}))
	}
	// Submit channel into actor control loop.
	// Should be effectively nonblocking, as long as the actor is started.
	// TODO explicitly panic about invalid state on reuse (currently hangs).
	r.rigging <- stream
	// Return immediately.
}
