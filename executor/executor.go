package executor

import (
	"polydawn.net/repeatr/def"
)

type Executor interface {
	Run(def.Formula) (def.Job, []def.Output)
}
