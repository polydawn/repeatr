package iofilter

import (
	"io"
	"sync"
)

var _ io.Writer = &SyncWriter{}

type SyncWriter struct {
	W  io.Writer
	mu sync.Mutex
}

func NewSyncWriter(w io.Writer) io.Writer {
	return &SyncWriter{W: w}
}

func (w *SyncWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.W.Write(b)
}
