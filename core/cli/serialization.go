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
	Level int       `json:"level"`
	Msg   string    `json:"message"`
	Ctx   ctxPairs  `json:"ctx"`
	Time  time.Time `json:"time"`
}

type runRecordOutput struct {
	Data  interface{} `json:"runRecord"`
	RunID def.RunID   `json:"runID"`
}
type journalOutput struct {
	Data  string    `json:"journal"`
	RunID def.RunID `json:"runID"`
}
type logOutput struct {
	Data logItem `json:"log"`
}

type journalSerializer struct {
	Delegate io.Writer
	RunID    def.RunID
}

func (a *journalSerializer) Write(b []byte) (int, error) {
	jo := journalOutput{Data: string(b), RunID: a.RunID}
	out, err := json.Marshal(jo)
	if err != nil {
		panic(err)
	}
	a.Delegate.Write(append(out, byte('\n')))
	return len(b), nil
}

func serializeRunRecord(wr io.Writer, runID def.RunID, i interface{}) error {
	err := codec.NewEncoder(wr, &codec.JsonHandle{}).Encode(runRecordOutput{
		Data:  i,
		RunID: runID,
	})
	wr.Write([]byte{'\n'})
	return err
}

func logHandler(wr io.Writer) log15.Handler {
	h := log15.FuncHandler(func(r *log15.Record) error {
		li := logItem{
			Level: int(r.Lvl),
			Msg:   r.Msg,
			Ctx:   r.Ctx,
			Time:  r.Time,
		}
		err := codec.NewEncoder(wr, &codec.JsonHandle{}).Encode(logOutput{
			Data: li,
		})
		wr.Write([]byte{'\n'})
		return err
	})
	return log15.LazyHandler(log15.SyncHandler(h))
}
