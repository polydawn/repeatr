package iofilter

import (
	"io"
	"io/ioutil"
)

/*
   StallingWriter wraps another io.Writer and presents it as the same,
   but after creation any write calls will block until the StallingWriter
   is instructed to release.
   When released, all blocked writes will proceed in their original order
   and all writes thereafter will proceed directly to the wrapped writer.

   (StallingWriter blocks outright as opposed to internally buffering,
   because internal buffering necessarily would require that error codes
   and lengths returned by `Write()` before the release are lies.  We
   prefer not to lie.)
*/
type StallingWriter struct {
	w     io.Writer
	queue chan struct{}
}

var _ io.Writer = &StallingWriter{}

func NewStallingWriter(w io.Writer) *StallingWriter {
	return &StallingWriter{
		w:     w,
		queue: make(chan struct{}, 0),
	}
}

func (w *StallingWriter) Write(msg []byte) (int, error) {
	<-w.queue
	return w.w.Write(msg)
}

/*
	Call to allow writes to proceed through to the wrapped writer.

	Call either this method or the 'Discard' method exactly once;
	repeated calls will panic (much like repeated 'close' on a channel).
*/
func (w *StallingWriter) Release() {
	close(w.queue)
}

/*
	Call to allow allow writes to proceed, but no-op them.

	Call either this method or the 'Release' method exactly once;
	repeated calls will panic (much like repeated 'close' on a channel).
*/
func (w *StallingWriter) Discard() {
	w.w = ioutil.Discard
	close(w.queue)
}
