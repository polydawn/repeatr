package mixins

import (
	"context"
	"sync"
	"time"

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
			switch evt2 := evt.(type) {
			case rio.Event_Log:
				mon.Chan <- repeatr.Event_Log{
					Time:   evt2.Time,
					Level:  repeatr.LogLevel(evt2.Level),
					Msg:    evt2.Msg,
					Detail: evt2.Detail,
				}
			case rio.Event_Progress:
				// pass... for now
			}
		case <-ctx.Done():
			return
		}
	}
}

/*
	Wraps a Rio stitch cleanup func to log any errors to the `repeatr.Event`
	channel.

	Typical usage is to defer this (just like you would the CleanupFunc).
*/
func CleanupFuncWithLogging(cleanupFunc func() error, mon repeatr.Monitor) func() {
	return func() {
		err := cleanupFunc()
		if err == nil {
			return
		}
		mon.Send(repeatr.Event_Log{
			Time:  time.Now(),
			Level: repeatr.LogError,
			Msg:   "error during cleanup: " + err.Error(),
			Detail: [][2]string{
				{"error", err.Error()},
			},
		})
	}
}
