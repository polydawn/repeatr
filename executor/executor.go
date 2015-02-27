package executor

import (
	"polydawn.net/repeatr/def"
)

type Executor interface {
	Run(def.JobRecord) (def.ActiveJob, []def.Output)
}
