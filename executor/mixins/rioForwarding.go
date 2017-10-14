package mixins

import (
	"context"
	"sync"

	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/rio/stitch"
)

/*
	Set `rio.Monitor`s on a bunch of unpackSpecs to all forward log events
	to the `repeatr.Monitor` log events.

	The arg slice contents are modified in place.

	A sync.WaitGroup is returned; make sure to `Wait()` for it in order
	to be sure all logs have been forwarded.
*/
func ForwardRioUnpackLogs(
	ctx context.Context,
	mon repeatr.Monitor,
	unpackSpecs []stitch.UnpackSpec,
) *sync.WaitGroup {
	var wg sync.WaitGroup
	if mon.Chan == nil {
		return &wg
	}
	for i, _ := range unpackSpecs {
		wg.Add(1)
		ch := make(chan rio.Event)
		unpackSpecs[i].Monitor = rio.Monitor{ch}
		go func() {
			defer wg.Done()
			forwardRioUnpackLogLoop(ctx, mon, ch)
		}()
	}
	return &wg
}

func forwardRioUnpackLogLoop(
	ctx context.Context,
	mon repeatr.Monitor,
	rioCh <-chan rio.Event,
) {
	for {
		select {
		case evt, ok := <-rioCh:
			if !ok {
				return
			}
			switch {
			case evt.Log != nil:
				mon.Chan <- repeatr.Event{Log: &repeatr.Event_Log{
					Time:   evt.Log.Time,
					Level:  repeatr.LogLevel(evt.Log.Level),
					Msg:    evt.Log.Msg,
					Detail: evt.Log.Detail,
				}}
			case evt.Progress != nil:
				// pass... for now
			}
		case <-ctx.Done():
			return
		}
	}
}
