package testutil

import (
	"io"

	"github.com/inconshreveable/log15"
	"github.com/smartystreets/goconvey/convey"
)

func TestLogger(c convey.C) log15.Logger {
	log := log15.New()
	log.SetHandler(log15.StreamHandler(Writer{c}, log15.TerminalFormat()))
	return log
}

var _ io.Writer = Writer{}

/*
	Wraps a goconvey context into an `io.Writer` so that you can
	shovel logs at it.

	... I really WANT goconvey to be clever about buffering this,
	aligning it reasonably, and shutting it up unless something goes
	wrong.  Alas, the terminal form of goconvey does none of these things.
*/
type Writer struct {
	Convey convey.C
}

func (lw Writer) Write(msg []byte) (int, error) {
	return lw.Convey.Print(string(msg))
}
