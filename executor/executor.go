package executor

import (
	"polydawn.net/repeatr/def"
)

type Executor interface {
	Configure(workspacePath string)

	/*

		Returns a channel that will asyncronously return a reference to a running Job.

		An invalid formula (as far as can be determined without hitting the filesystem) will panic before returning.
		Once this function has returned, the executor is presumed to be spinning up the job in a separate goroutine.

		The executor expects to be configured with a workspace path, and will create a directory inside that, probably with a job GUID.
		It is assumed that any job-specific filesystem state will be cleaned up by the executor.

	*/
	Start(def.Formula) def.Job
}
