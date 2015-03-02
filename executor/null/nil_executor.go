package null

import (
	"polydawn.net/repeatr/def"
)

type Executor struct {
}

func (*Executor) Run(job def.Formula) (def.Job, []def.Output) {
	return nil, nil
}
