package def

import (
	"time"
)

type ctxPairs []interface{}

func (ctxPairs) MapBySlice() {}

type LogItem struct {
	Level int       `json:"lvl"`
	Msg   string    `json:"msg"`
	Ctx   ctxPairs  `json:"ctx"`
	Time  time.Time `json:"t"`
}

type SerializedOutput struct {
	RunID     RunID      `json:"runID,omitempty"`
	RunRecord *RunRecord `json:"runRecord,omitempty"`
	Journal   string     `json:"journal,omitempty"`
	Log       *LogItem   `json:"log,omitempty"`
}
