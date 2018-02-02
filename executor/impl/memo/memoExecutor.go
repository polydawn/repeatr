package memo

import (
	"context"
	"time"

	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/rio/fs"
)

type Executor struct {
	memoDir  fs.AbsolutePath
	delegate repeatr.RunFunc
}

func NewExecutor(
	memoDir fs.AbsolutePath,
	delegate repeatr.RunFunc,
) (repeatr.RunFunc, error) {
	return Executor{
		memoDir, delegate,
	}.Run, nil
}

var _ repeatr.RunFunc = Executor{}.Run

func (cfg Executor) Run(
	ctx context.Context,
	formula api.Formula,
	formulaCtx api.FormulaContext,
	input repeatr.InputControl,
	mon repeatr.Monitor,
) (_ *api.RunRecord, err error) {
	defer RequireErrorHasCategory(&err, repeatr.ErrorCategory(""))

	// Consider possibility of early return of memoization data.
	//  If a memo dir is set and it contains a relevant record, we just echo it.
	rr, err := loadMemo(formula.SetupHash(), cfg.memoDir)
	if err != nil {
		return nil, err
	}
	if rr != nil {
		mon.Chan <- repeatr.Event{Log: &repeatr.Event_Log{
			Time:  time.Now(),
			Level: repeatr.LogInfo,
			Msg:   "memoized runRecord found for formula setupHash; eliding run",
			Detail: [][2]string{
				{"setupHash", string(formula.SetupHash())},
			},
		}}
		return rr, nil
	}

	// If no shortcut: delegate to the real executor to do work!
	rr, err = cfg.delegate(ctx, formula, formulaCtx, input, mon)

	// Save memo for next time (unless there was an executor error).
	if err == nil {
		if err := saveMemo(formula.SetupHash(), cfg.memoDir, rr); err != nil {
			mon.Chan <- repeatr.Event{Log: &repeatr.Event_Log{
				Time:  time.Now(),
				Level: repeatr.LogWarn,
				Msg:   "saving memoized runRecord failed",
				Detail: [][2]string{
					{"err", err.Error()},
				},
			}}
		}
	}

	return rr, err
}
