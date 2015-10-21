package streamer

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

var _ io.ReadCloser = &TailReader{}

var ErrAlreadyClosed = fmt.Errorf("TailReader already closed")

type TailReader struct {
	r     io.Reader
	drain int32
}

/*
	Proxies another reader, disregarding EOFs and blocking instead until
	the user closes.
*/
func NewTailReader(r io.Reader) *TailReader {
	return &TailReader{
		r: r,
	}
}

func (r *TailReader) Read(msg []byte) (n int, err error) {
	for n == 0 && err == nil {
		n, err = r.r.Read(msg)
		if err == io.EOF {
			// We don't pass EOF up until we're commanded to be closed.
			// This could be a "temporary" EOF and appends will still be incoming.
			if n > 0 {
				// If any bytes, pass them up immediately.
				return n, nil
			}
			// If we got EOF, have no buffer, and are at this instant closed, leave.
			if r.drain > 0 {
				return 0, io.EOF
			}
			// Pause before retrying.
			// We're effectively required to block here, because otherwise the reader may spin;
			// this is not a clueful wait; but it does prevent pegging a core.
			// Quite dumb in this case is also quite fool-proof.
			err = nil
			<-time.After(1 * time.Millisecond)
		}
	}
	// anything other than an eof, we have no behavioral changes to make; pass up.
	return n, err
}

/*
	Breaks any readers currently blocked.
*/
func (r *TailReader) Close() error {
	if swapped := atomic.CompareAndSwapInt32(&r.drain, 0, 1); swapped != true {
		return ErrAlreadyClosed
	}
	return nil
}
