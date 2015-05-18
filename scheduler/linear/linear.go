package linear

import (
	"io"

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
	executor         executor.Executor
	queue            chan *hold
	jobLoggerFactory func(def.JobID) io.Writer
}

func (s *Scheduler) Configure(e executor.Executor, queueSize int, jobLoggerFactory func(def.JobID) io.Writer) {
	s.executor = e
	s.queue = make(chan *hold, queueSize)
	s.jobLoggerFactory = jobLoggerFactory
}

func (s *Scheduler) Start() {
	go s.Run()
}

func (s *Scheduler) Schedule(f def.Formula) (def.JobID, <-chan def.Job) {
	id := def.JobID(guid.New())

	h := &hold{
		id:       id,
		forumla:  f,
		response: make(chan def.Job),
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
		job := s.executor.Start(h.forumla, h.id, journal)
		h.response <- job
		job.Wait()
	}
}
