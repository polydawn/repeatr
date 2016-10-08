package runner

import (
	"io"

	"github.com/inconshreveable/log15"

	"go.polydawn.net/repeatr/api/def"
)

type evtStreamLogHandler struct {
	runID  def.RunID
	stream chan<- *def.Event
}

var _ log15.Handler = &evtStreamLogHandler{}

func (x *evtStreamLogHandler) Log(r *log15.Record) error {
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
}

func (x *evtStreamLogHandler) NewLogger() log15.Logger {
	log := log15.New()
	log.SetHandler(x)
	return log
}

var _ io.Writer = &evtStreamJournalWriter{}

type evtStreamJournalWriter struct {
	runID  def.RunID
	stream chan<- *def.Event
}

func (w *evtStreamJournalWriter) Write(b []byte) (int, error) {
	w.stream <- &def.Event{
		RunID:   w.runID,
		Journal: string(b),
	}
	return len(b), nil
}
