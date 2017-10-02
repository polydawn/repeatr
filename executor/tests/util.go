package tests

import (
	"bytes"

	"go.polydawn.net/go-timeless-api/repeatr"
)

type bufferingMonitor struct {
	Txt bytes.Buffer
	Ch  chan repeatr.Event
	Err error
}

func (bm bufferingMonitor) init() *bufferingMonitor {
	bm = bufferingMonitor{
		Ch: make(chan repeatr.Event),
	}
	go func() { // leaks.  fuck the police.
		for {
			bm.Err = repeatr.CopyOut(<-bm.Ch, &bm.Txt)
		}
	}()
	return &bm
}
func (bm *bufferingMonitor) monitor() repeatr.Monitor {
	return repeatr.Monitor{Chan: bm.Ch}
}
