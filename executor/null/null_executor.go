package null

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/basicjob"
)

type Executor struct {
}

func (*Executor) Configure(workspacePath string) {
}

func (*Executor) Start(f def.Formula) def.Job {
	job := basicjob.New()

	go func() {
		close(job.WaitChan)
	}()

	return job
}
