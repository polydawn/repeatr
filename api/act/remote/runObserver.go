package remote

import (
	"bytes"
	"io"
	"strings"

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

	// Keep the last partial message decode here, for dumping in error cases.
	replay bytes.Buffer
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
	roc.replay.Reset()
	r := io.TeeReader(roc.Remote, &roc.replay)
	err := codec.NewDecoder(r, roc.Codec).Decode(&evt)
	meep.TryPlan{
		{ByVal: io.EOF,
			Handler: func(error) {
				eof = true
			}},
		{CatchAny: true,
			Handler: func(e error) {
				// Read out the rest.
				// Up until some fairly high limit,anyway.
				io.CopyN(&roc.replay, roc.Remote, 1024*1024)
				// Trim.
				// This is a lossy conversion, but we're already
				// subscribing to a belief that this is gonna be a
				// human-readable string, so cleanup is fair game.
				dump := strings.TrimSpace(roc.replay.String())
				panic(meep.Meep(
					&act.ErrRemotePanic{Dump: dump},
					meep.Cause(e),
				))
			}},
	}.MustHandle(err)
	return
}
