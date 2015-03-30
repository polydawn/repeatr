package linear

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/scheduler"
)

// interface assertion
var _ scheduler.Scheduler = &Scheduler{}

type Scheduler struct {
	executor *executor.Executor
	queue    chan def.Formula
	results  chan def.Job
}

func (s *Scheduler) Configure(e *executor.Executor) {
	s.executor = e
	s.queue = make(chan def.Formula)
	s.results = make(chan def.Job)
}

func (s *Scheduler) Start() {
	go s.Run()
}

func (s *Scheduler) Queue() chan<- def.Formula {
	return s.queue
}

func (s *Scheduler) Results() <-chan def.Job {
	return s.results
}

// Run jobs one at a time
func (s *Scheduler) Run() {
	for f := range s.queue {
		job := (*s.executor).Start(f)
		s.results <- job
		job.Wait()
	}
}
