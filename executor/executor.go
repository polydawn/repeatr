package executor

import (
	"polydawn.net/repeatr/def"
)

type Executor interface {
	Configure(workspacePath string)

	/*

		The executor expects to be configured with a workspace path, and will create a directory inside that, probably with a job GUID.
		It is assumed that any job-specific filesystem state will be cleaned up by the executor.

	*/
	Run(def.Formula) (def.Job, []def.Output)
}
