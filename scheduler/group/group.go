package group

import (
	"runtime"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/scheduler"
)

// interface assertion
var _ scheduler.Scheduler = &Scheduler{}

// Dumb struct to send job references back
type hold struct {
	id       def.JobID
	forumla  def.Formula
	response chan def.Job
}

type Scheduler struct {
	groupSize int
	executor  *executor.Executor
	queue     chan *hold
}

func (s *Scheduler) Configure(e *executor.Executor) {
	s.groupSize = runtime.NumCPU()
	s.executor = e
	s.queue = make(chan *hold)
}

func (s *Scheduler) Start() {
	for w := 1; w <= s.groupSize; w++ {
		go s.Run()
	}
}

func (s *Scheduler) Schedule(f def.Formula) (def.JobID, <-chan def.Job) {
	id := def.JobID(guid.New())

	h := &hold{
		id:       id,
		forumla:  f,
		response: make(chan def.Job),
	}

	go func() {
		s.queue <- h
	}()

	return id, h.response
}

// Run jobs one at a time
func (s *Scheduler) Run() {
	for h := range s.queue {

		job := (*s.executor).Start(h.forumla, h.id)
		h.response <- job
		job.Wait()
	}
}
