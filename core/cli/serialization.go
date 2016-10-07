package cli

import (
	"encoding/json"
	"io"

	"go.polydawn.net/repeatr/api/def"

	"github.com/inconshreveable/log15"
	"github.com/ugorji/go/codec"
)

var _ io.Writer = &journalSerializer{}

type journalSerializer struct {
	Delegate io.Writer
	RunID    def.RunID
}

func (a *journalSerializer) Write(b []byte) (int, error) {
	jo := def.Event{Journal: string(b), RunID: a.RunID}
	out, err := json.Marshal(jo)
	if err != nil {
		panic(err)
	}
	a.Delegate.Write(append(out, byte('\n')))
	return len(b), nil
}

func logHandler(wr io.Writer) log15.Handler {
	h := log15.FuncHandler(func(r *log15.Record) error {
		li := &def.LogItem{
			Level: int(r.Lvl),
			Msg:   r.Msg,
			Ctx:   r.Ctx,
			Time:  r.Time,
		}
		err := codec.NewEncoder(wr, &codec.JsonHandle{}).Encode(def.Event{Log: li})
		wr.Write([]byte{'\n'})
		return err
	})
	return log15.LazyHandler(log15.SyncHandler(h))
}
