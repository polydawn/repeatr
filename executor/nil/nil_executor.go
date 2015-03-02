package nil

import (
	"polydawn.net/repeatr/def"
)

type Executor struct {
}

func (*Executor) Run(job def.JobDraft) (def.Job, []def.Output) {
	return nil, nil
}
