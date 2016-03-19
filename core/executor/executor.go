package executor

import (
	"io"

	"polydawn.net/repeatr/def"
)

/*
	In general, executors are assumed to be running on the host that will run the Forumla.

	Coordinating remote hosts is generally the responsibiltiy of a scheduler.
*/
type Executor interface {

	/*
		Prepare the Executor with information it needs to run.
	*/
	Configure(workspacePath string)

	/*

		Validates the passed forumla and returns a Job that is or will soon be running.

		An invalid formula (as far as can be determined without hitting the filesystem) will panic before returning.
		Once this function has returned, the executor is presumed to be spinning up the job in a separate goroutine.

		The executor expects to be configured with a workspace path, and will create a directory inside that, probably with a job GUID.
		It is assumed that any job-specific filesystem state will be cleaned up by the executor.

	*/
	Start(def.Formula, def.JobID, io.Reader, io.Writer) def.Job

	/*
		ADDITIONALLY, we have some patterns that are merely conventions:


		// Executes a job, catching any panics.
		func (e *Executor) Run(f def.Formula, j def.Job, d string) def.JobResult {

		// Execute a forumla in a specified directory. MAY PANIC.
		func (e *Executor) Execute(f def.Formula, j def.Job, d string) def.JobResult {


		An executor should absolutely not be tied down, so leaving these implicit for now.
		If your executor CAN follow this pattern, that would be good.
	*/
}
