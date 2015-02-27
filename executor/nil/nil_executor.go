package nil

import (
	"polydawn.net/repeatr/def"
)

type Executor struct {
}

func (*Executor) Run(job def.JobRecord) (def.ActiveJob, []def.Output) {
	return nil, nil
}
