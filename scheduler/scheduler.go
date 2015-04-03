package scheduler

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
)

/*
	"No one has ever looked at a cloud scheduler and thought, 'this is everything I need!'".

	Schedulers manage a stream of Formulas and return running Jobs.
	These Jobs might be scheduled locally or over an enormous cluster of remote machines.

	Because these imply significant complexity, it would be naive to assume that a scheduler interface could be one-size-fits-all.
	This interface comprises an optimistic starting point that schedulers can follow, but may surpass.

	Schedulers are presumed to know environmental context that a Formula provider may not.
	Any problems with a transport running on a remote host, for example, is the scheduler's problem.

	REVIEW: our interfaces do not support returning a guid from scheduling, as executors create their guid when starting.
	It would be cleaner to be able to refer to a scheduled Job without waiting arbitrarily long for the scheduler to start the job.
*/
type Scheduler interface {

	/*
		Configure executor to use. Must be already configured.

		It is guaranteed that calling Use() before scheduling work will behave as expected.
		Calling Use() after scheduling work is left for the Scheduler to decide - it might change, panic, ignore, etc.
	*/
	Configure(*executor.Executor)

	/*
		Start consuming Formulas.
		It is expected that you call Configure(), then Start(), before scheduling Formulas.
	*/
	Start()

	/*
		Schedules a Forumla to be ran, and returns a channel that will hand you a Job.
	*/
	Schedule(def.Formula) <-chan def.Job
}

/*
	ADDITIONALLY, we have some patterns that are merely conventions:

	// The run loop, which is ran in a dedicated goroutine when Start() is called.
	func (s Scheduler) Run() {
*/
