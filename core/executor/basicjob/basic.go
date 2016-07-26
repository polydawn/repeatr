package basicjob

import (
	"io"

	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/lib/streamer"
)

type BasicJob struct {
	ID executor.JobID

	Streams streamer.ROMux

	// Only valid to read after Wait()
	Result executor.JobResult

	// This channel should never be sent to, and is instead closed when the job is complete.
	WaitChan chan struct{}
}

func (j *BasicJob) Id() executor.JobID {
	return j.ID
}

func (j *BasicJob) OutputReader() io.Reader {
	return j.Streams.Reader(1, 2)
}

func (j *BasicJob) Outputs() streamer.ROMux {
	return j.Streams
}

func (j *BasicJob) Wait() executor.JobResult {
	<-j.WaitChan
	return j.Result
}

func New(id executor.JobID) *BasicJob {
	return &BasicJob{
		ID:       id,
		WaitChan: make(chan struct{}),
	}
}
