package null

import (
	"polydawn.net/repeatr/def"
)

type Executor struct {
}

func (*Executor) Configure(workspacePath string) {
}

func (*Executor) Run(job def.Formula) (def.Job, []def.Output) {
	return nil, nil
}
