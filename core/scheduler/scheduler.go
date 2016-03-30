package scheduler

import (
	"io"

	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/def"
)

/*
	"No one has ever looked at a cloud scheduler and thought, 'this is everything I need!'".

	Schedulers manage a stream of Formulas and return running Jobs.
	These Jobs might be scheduled locally or over an enormous cluster of remote machines.

	Because these imply significant complexity, it would be naive to assume that a scheduler interface could be one-size-fits-all.
	This interface comprises an optimistic starting point that schedulers can follow, but may surpass.

	Schedulers are presumed to know environmental context that a Formula provider may not.
	Any problems with a transport running on a remote host, for example, is the scheduler's problem.
*/
type Scheduler interface {

	/*
		Configure executor to use. Must be already configured.

		It is guaranteed that calling Use() before scheduling work will behave as expected.
		Calling Use() after scheduling work is left for the Scheduler to decide - it might change, panic, ignore, etc.
	*/
	Configure(e executor.Executor, queueSize int, jobLoggerFactory func(def.JobID) io.Writer)

	/*
		Start consuming Formulas.
		It is expected that you call Configure(), then Start(), before scheduling Formulas.
	*/
	Start()

	/*
		Schedules a Forumla to run; returns the job ID and a channel that will hand you a Job instance.
	*/
	Schedule(def.Formula) (def.JobID, <-chan def.Job)
}

/*
	ADDITIONALLY, we have some patterns that are merely conventions:

	// The run loop, which is ran in a dedicated goroutine when Start() is called.
	func (s Scheduler) Run() {
*/
