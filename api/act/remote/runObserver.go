package remote

import (
	"io"

	"github.com/ugorji/go/codec"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/act"
	"go.polydawn.net/repeatr/api/def"
)

var _ act.RunObserver = &RunObserverClient{}

/*
	A read-only client that can be wrapped around an event stream pushed
	by e.g. `repeatr//api/act/remote/server.RunObserverServer`.
*/
type RunObserverClient struct {
	Remote io.Reader
	Codec  codec.Handle
}

func (roc *RunObserverClient) FollowEvents(
	which def.RunID,
	stream chan<- *def.Event,
	startingFrom def.EventSeqID,
) {
	// TODO this should probably accept a Supervisor so it's interruptable.
	// TODO we're totally disregarding `startingFrom` right now.

	for {
		evt, eof := roc.readOne()
		if eof {
			break
		}
		stream <- &evt
	}
}

func (roc *RunObserverClient) readOne() (evt def.Event, eof bool) {
	err := codec.NewDecoder(roc.Remote, roc.Codec).Decode(&evt)
	meep.TryPlan{
		{ByVal: io.EOF,
			Handler: func(error) {
				eof = true
			}},
		{CatchAny: true,
			Handler: func(error) {
				panic(meep.Meep(&act.ErrRemotePanic{Dump: "todo"}))
			}},
	}.MustHandle(err)
	return
}
