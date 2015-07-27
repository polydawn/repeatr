package null

import (
	"io"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/basicjob"
)

// interface assertion
var _ executor.Executor = &Executor{}

type Executor struct {
}

func (*Executor) Configure(workspacePath string) {
}

func (*Executor) Start(f def.Formula, id def.JobID, stdin io.Reader, journal io.Writer) def.Job {
	job := basicjob.New(id)

	go func() {
		close(job.WaitChan)
	}()

	return job
}
