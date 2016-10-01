package def

import (
	"time"
)

/*
	A "union" type of all the kinds of event that may be generated in the
	course of running a formula.
*/
type Event struct {
	RunID RunID      `json:"runID"`
	Seq   EventSeqID `json:"seq"`

	// Union fields -- only one should be set:

	// Log events are repeatr's logs.
	Log *LogItem `json:"log,omitempty"`

	// User output strings are called journal events.
	Journal string `json:"journal,omitempty"`

	// RunRecords are the final event emitted by a run.
	RunRecord *RunRecord `json:"runRecord,omitempty"`
}

type EventSeqID int

/*
	Type translating log15 events for serialization in `Event`s.
*/
type LogItem struct {
	Time  time.Time `json:"t"`
	Level int       `json:"lvl"`
	Msg   string    `json:"msg"`
	Ctx   ctxPairs  `json:"ctx"`
}

type ctxPairs []interface{}

func (ctxPairs) MapBySlice() {}
