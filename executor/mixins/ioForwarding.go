package mixins

import (
	"context"
	"io"
	"io/ioutil"
	"time"

	"go.polydawn.net/go-timeless-api/repeatr"
)

/*
	Returns an `io.Writer` which proxies each `Write` call
	into a `repeatr.Event_Output` and fires it into the channel.

	If given a nil channel, the returned writer will be ioutil.Discard
	(so yes, you can use it on `repeatr.Monitor.Chan` without even looking).
*/
func NewOutputEventWriter(ctx context.Context, ch chan<- repeatr.Event) io.Writer {
	if ch == nil {
		return ioutil.Discard
	}
	return chanWriter{ctx, ch}
}

type chanWriter struct {
	ctx context.Context
	ch  chan<- repeatr.Event
}

func (chw chanWriter) Write(bs []byte) (int, error) {
	select {
	case chw.ch <- repeatr.Event_Output{
		Time: time.Now(),
		Msg:  string(bs),
	}: // nice
	case <-chw.ctx.Done():
		return 0, nil
	}
	return len(bs), nil
}

/*
	Proxies each item in the channel into a call to the given `io.WriteCloser`.
	The writer is closed when the channel is closed.
	(Pairs well with `cmd.StdinPipe`.)
*/
func RunInputWriteForwarder(ctx context.Context, writeTo io.WriteCloser, ch <-chan string) {
	go func() {
		for {
			select {
			case chunk, ok := <-ch:
				if !ok {
					writeTo.Close()
					return
				}
				writeTo.Write([]byte(chunk))
			case <-ctx.Done():
				writeTo.Close()
				return
			}
		}
	}()
}
