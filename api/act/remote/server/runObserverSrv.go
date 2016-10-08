package server

import (
	"io"

	"github.com/ugorji/go/codec"
	"go.polydawn.net/go-sup"

	"go.polydawn.net/repeatr/api/act"
	"go.polydawn.net/repeatr/api/def"
)

type RunObserverPublisher struct {
	// When run, the publisher will read this.
	//  Note we don't use the full `act.RunObserver` because we're a
	//  uni-directional implementation; there's no point in having the
	//  full resume/seek API at hand if we aren't going to use it.
	Proxy act.RunObserver

	// The RunID to proxy the RunObserver's events about.
	RunID def.RunID

	// When run, the publisher will write the serialized event stream to this.
	//  Note that this is uni-directional; the far side will not be able
	//  to do any resume/seek if our io stream fails.
	Output io.Writer

	// This byte slice will be written to Output after every event, if set.
	//  (This is useful for inserting "\n" betwen json, for example.)
	RecordSeparator []byte

	// Set this codec to control how the publisher serializes events.
	Codec codec.Handle

	// cache.
	encoder *codec.Encoder

	// set at start of run when we subscribe to the observer.
	evtStream chan *def.Event
}

func (a *RunObserverPublisher) Run(supvr sup.Supervisor) {
	a.encoder = codec.NewEncoder(a.Output, a.Codec)
	a.evtStream = make(chan *def.Event)
	go a.Proxy.FollowEvents(a.RunID, a.evtStream, 0) // this badly does need a supervisor
	for a.step(supvr) {
	}
}

func (a *RunObserverPublisher) step(supvr sup.Supervisor) (more bool) {
	select {
	case evt := <-a.evtStream:
		a.encoder.Encode(evt)
		a.Output.Write(a.RecordSeparator)
		return evt.RunRecord == nil
	case <-supvr.QuitCh():
		return false
	}
}
