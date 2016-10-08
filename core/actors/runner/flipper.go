package runner

import (
	"io"

	"github.com/inconshreveable/log15"

	"go.polydawn.net/repeatr/api/def"
)

/*
	Implements^W ALMOST implements act.RunObserver
	and manufactures the sinks (the logger, journal, etc)
	to hand to the executor.
*/
type flipper struct {
	runID  def.RunID
	stream chan<- *def.Event
}

func (x *flipper) GetLogger() log15.Logger {
	log := log15.New()
	log.SetHandler(log15.FuncHandler(func(r *log15.Record) error {
		x.stream <- &def.Event{
			RunID: x.runID,
			Log: &def.LogItem{
				Level: int(r.Lvl),
				Msg:   r.Msg,
				Ctx:   r.Ctx,
				Time:  r.Time,
			},
		}
		return nil
	}))
	return log
}

type journalEventWriter struct {
	runID  def.RunID
	stream chan<- *def.Event
}

func (w *journalEventWriter) Write(b []byte) (int, error) {
	w.stream <- &def.Event{
		RunID:   w.runID,
		Journal: string(b),
	}
	return len(b), nil
}

func (x *flipper) GetJournal() io.Writer {
	return &journalEventWriter{x.runID, x.stream}
}

func (x *flipper) Result(rr *def.RunRecord) {
	x.stream <- &def.Event{
		RunID:     x.runID,
		RunRecord: rr,
	}
}
