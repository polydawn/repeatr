package linear

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/scheduler"
)

// interface assertion
var _ scheduler.Scheduler = &Scheduler{}

// Dumb struct to send job references back
type hold struct {
	forumla def.Formula
	response chan def.Job
}

type Scheduler struct {
	executor *executor.Executor
	queue    chan *hold
}

func (s *Scheduler) Configure(e *executor.Executor) {
	s.executor = e
	s.queue = make(chan *hold)
}

func (s *Scheduler) Start() {
	go s.Run()
}

func (s *Scheduler) Schedule(f def.Formula) <-chan def.Job {

	h := &hold{
		forumla: f,
		response: make(chan def.Job),
	}

	go func() {
		s.queue <- h
	}()

	return h.response
}

// Run jobs one at a time
func (s *Scheduler) Run() {
	for h := range s.queue {
		job := (*s.executor).Start(h.forumla)
		h.response <- job
		job.Wait()
	}
}
