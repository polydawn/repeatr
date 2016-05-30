package linear

import (
	"io"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/core/scheduler"
	"polydawn.net/repeatr/lib/guid"
)

// interface assertion
var _ scheduler.Scheduler = &Scheduler{}

// Dumb struct to send job references back
type hold struct {
	id       executor.JobID
	forumla  def.Formula
	response chan executor.Job
}

type Scheduler struct {
	executor         executor.Executor
	queue            chan *hold
	jobLoggerFactory func(executor.JobID) io.Writer
}

func (s *Scheduler) Configure(e executor.Executor, queueSize int, jobLoggerFactory func(executor.JobID) io.Writer) {
	s.executor = e
	s.queue = make(chan *hold, queueSize)
	s.jobLoggerFactory = jobLoggerFactory
}

func (s *Scheduler) Start() {
	go s.Run()
}

func (s *Scheduler) Schedule(f def.Formula) (executor.JobID, <-chan executor.Job) {
	id := executor.JobID(guid.New())

	h := &hold{
		id:       id,
		forumla:  f,
		response: make(chan executor.Job),
	}

	// Non-blocking send, will panic if scheduler queue is full
	select {
	case s.queue <- h:
	default:
		panic(scheduler.QueueFullError)
	}

	return id, h.response
}

// Run jobs one at a time
func (s *Scheduler) Run() {
	for h := range s.queue {
		journal := s.jobLoggerFactory(h.id) // TODO replace with real logging framework
		job := s.executor.Start(h.forumla, h.id, nil, journal)
		h.response <- job
		job.Wait()
	}
}
