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
		evt := roc.readOne()
		if evt == (def.Event{}) {
			break
		}
		stream <- &evt
	}
}

func (roc *RunObserverClient) AwaitRunRecord(def.RunID) *def.RunRecord {
	return nil // TODO
	// REVIEW: question whether this should be on this interface at all.
	// It's something you can generate by watching the stream;
	// it makes no sense to have every stream implementation watch for it itself.
}

func (roc *RunObserverClient) readOne() def.Event {
	out := def.Event{}

	err := codec.NewDecoder(roc.Remote, roc.Codec).Decode(&out)
	meep.TryPlan{
		{ByVal: io.EOF,
			Handler: meep.TryHandlerDiscard},
		{CatchAny: true,
			Handler: func(error) {
				panic(meep.Meep(&act.ErrRemotePanic{Dump: "todo"}))
			}},
	}.MustHandle(err)

	return out
}
