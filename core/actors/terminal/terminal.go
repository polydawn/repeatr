/*
	Consume `act.RunObserver` and prints results to a standard IO streams,
	returning the RunRecord when done.
*/
package terminal

import (
	"io"

	"github.com/inconshreveable/log15"

	"go.polydawn.net/repeatr/api/act"
	"go.polydawn.net/repeatr/api/def"
)

func Consume(observationPost act.RunObserver, runID def.RunID, stdout, stderr io.Writer) *def.RunRecord {
	evtStream := make(chan *def.Event)
	go observationPost.FollowEvents(runID, evtStream, 0)
	for {
		if rr := step(evtStream, stdout, stderr); rr != nil {
			return rr
		}
	}
}

func step(evtStream chan *def.Event, stdout, stderr io.Writer) *def.RunRecord {
	select {
	case evt := <-evtStream:
		if evt.Log != nil {
			// Shell out to log15 formatters.  Could do more custom.
			stderr.Write(log15.TerminalFormat().Format(&log15.Record{
				Time: evt.Log.Time,
				Lvl:  log15.Lvl(evt.Log.Level),
				Msg:  evt.Log.Msg,
				Ctx:  evt.Log.Ctx,
			}))
			return nil
		}
		if evt.Journal != "" {
			// FIXME journal entries should just be byte slices
			stdout.Write([]byte(evt.Journal))
			return nil
		}
		if evt.RunRecord != nil {
			return evt.RunRecord
		}
		panic("incomprehensible event")
	}
}
