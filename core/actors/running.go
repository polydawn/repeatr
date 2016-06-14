package actor

import (
	"io"
	"sync"
	"time"

	"github.com/inconshreveable/log15"

	"polydawn.net/repeatr/api/act"
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/lib/guid"
)

var (
	_ act.StartRun      = (&FormulaRunnerConfig{}).StartRun
	_ act.FollowStreams = (&FormulaRunnerConfig{}).FollowStreams
	_ act.FollowResults = (&FormulaRunnerConfig{}).FollowResults
)

type FormulaRunnerConfig struct {
	// config

	executor executor.Executor
	log      log15.Logger
	strmIn   io.Reader // only for 'twerk'.

	// state

	wards map[def.RunID]executor.Job
}

func NewFormulaRunner(
	execr executor.Executor,
	log log15.Logger,
) *FormulaRunnerConfig {
	return &FormulaRunnerConfig{
		executor: execr,
		log:      log,
		wards:    make(map[def.RunID]executor.Job),
	}
}

func (frCfg *FormulaRunnerConfig) InjectStdin(r io.Reader) {
	frCfg.strmIn = r
}

func (frCfg *FormulaRunnerConfig) StartRun(frm *def.Formula) def.RunID {
	// Assign arbitrary run id
	runID := def.RunID(guid.New())
	log := frCfg.log.New("runID", runID)

	// Give work the the executor.
	//  Returns a promise; execution goes off in parallel.
	log.Info("Formula evaluation Starting")
	job := frCfg.executor.Start(
		*frm,
		executor.JobID(runID),
		frCfg.strmIn,
		log,
	)
	frCfg.wards[runID] = job

	return runID
}

func (frCfg *FormulaRunnerConfig) FollowStreams(runID def.RunID, stdout io.Writer, stderr io.Writer) {
	job := frCfg.wards[runID]
	var wg sync.WaitGroup
	if stdout == stderr {
		wg.Add(1)
		go func() {
			io.Copy(stdout, job.Outputs().Reader(1, 2))
			wg.Done()
		}()
	} else {
		wg.Add(2)
		go func() {
			io.Copy(stdout, job.Outputs().Reader(1))
			wg.Done()
		}()
		go func() {
			io.Copy(stderr, job.Outputs().Reader(2))
			wg.Done()
		}()
	}
	wg.Wait()
}

func (frCfg *FormulaRunnerConfig) FollowResults(runID def.RunID) *def.RunRecord {
	job := frCfg.wards[runID]
	jr := job.Wait()
	results := def.ResultGroup{}
	for name, output := range jr.Outputs {
		results[name] = &def.Result{name, def.Ware{output.Type, output.Hash}}
	}
	return &def.RunRecord{
		UID:        def.RunID(job.Id()),
		Date:       time.Now(), // FIXME elide this translation layer, this should be committed just once
		FormulaHID: "todo",     // FIXME write formula HID
		Results:    results,
	}
}
