package basicjob

import (
	"io"

	"polydawn.net/repeatr/def"
)

type BasicJob struct {
	ID def.JobID

	Reader io.Reader

	// Only valid to read after Wait()
	Result def.JobResult

	// This channel should never be sent to, and is instead closed when the job is complete.
	WaitChan chan struct{}
}

func (j *BasicJob) Id() def.JobID {
	return j.ID
}

func (j *BasicJob) OutputReader() io.Reader {
	return j.Reader
}

func (j *BasicJob) Wait() def.JobResult {
	<-j.WaitChan
	return j.Result
}

func New(id def.JobID) *BasicJob {
	return &BasicJob{
		ID:       id,
		WaitChan: make(chan struct{}),
	}
}
