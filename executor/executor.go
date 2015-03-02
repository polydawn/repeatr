package executor

import (
	"polydawn.net/repeatr/def"
)

type Executor interface {
	Run(def.JobDraft) (def.Job, []def.Output)
}
