package cli

import (
	"encoding/json"
	"io"
	"time"

	"go.polydawn.net/repeatr/api/def"

	"github.com/inconshreveable/log15"
	"github.com/ugorji/go/codec"
)

var _ io.Writer = &journalSerializer{}

type ctxPairs []interface{}

func (ctxPairs) MapBySlice() {}

type logItem struct {
	Level int       `json:"lvl"`
	Msg   string    `json:"msg"`
	Ctx   ctxPairs  `json:"ctx"`
	Time  time.Time `json:"t"`
}

type serializedOutput struct {
	RunID     def.RunID      `json:"runID,omitempty"`
	RunRecord *def.RunRecord `json:"runRecord,omitempty"`
	Journal   string         `json:"journal,omitempty"`
	Log       *logItem       `json:"log,omitempty"`
}

type journalSerializer struct {
	Delegate io.Writer
	RunID    def.RunID
}

func (a *journalSerializer) Write(b []byte) (int, error) {
	jo := serializedOutput{Journal: string(b), RunID: a.RunID}
	out, err := json.Marshal(jo)
	if err != nil {
		panic(err)
	}
	a.Delegate.Write(append(out, byte('\n')))
	return len(b), nil
}

func serializeRunRecord(wr io.Writer, runID def.RunID, rr *def.RunRecord) error {
	err := codec.NewEncoder(wr, &codec.JsonHandle{}).Encode(serializedOutput{
		RunRecord: rr,
		RunID:     runID,
	})
	wr.Write([]byte{'\n'})
	return err
}

func logHandler(wr io.Writer) log15.Handler {
	h := log15.FuncHandler(func(r *log15.Record) error {
		li := &logItem{
			Level: int(r.Lvl),
			Msg:   r.Msg,
			Ctx:   r.Ctx,
			Time:  r.Time,
		}
		err := codec.NewEncoder(wr, &codec.JsonHandle{}).Encode(serializedOutput{Log: li})
		wr.Write([]byte{'\n'})
		return err
	})
	return log15.LazyHandler(log15.SyncHandler(h))
}
