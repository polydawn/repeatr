package actor

import (
	"io"

	"github.com/inconshreveable/log15"

	"polydawn.net/repeatr/api/act"
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/lib/guid"
)

var _ act.FormulaRunner = (&FormulaRunnerConfig{}).Run

type FormulaRunnerConfig struct {
	executor executor.Executor
	log      log15.Logger
	strmIn   io.Reader
	strmOut  io.Writer
	strmErr  io.Writer
}

/*
	REVIEW: there's a certain nice purity to the "formula-[run]->runrecord" definition,
	but for someone wanting to watch in near realtime, it's not exposing nearly enough.
	And if you imagine this being used in a farm where drilldown happens after launch,
	this idea of streams like this is... rong.  (Which is why the existing job
	promise looks the way it does.)
*/

func (frCfg *FormulaRunnerConfig) Run(frm *def.Formula) *def.RunRecord {
	// Assign arbitrary job id
	jobID := executor.JobID(guid.New())
	log := frCfg.log.New("jobID", jobID)

	// Give work the the executor.
	//  Returns a promise; execution goes off in parallel.
	log.Info("Formula evaluation Starting")
	job := frCfg.executor.Start(
		*frm,
		jobID,
		frCfg.strmIn,
		log,
	)

	// Stream input and outputs, if wired.
	// TODO

	// Wait for completion.  Log results before return.
	result := job.Wait()
	if result.Error != nil {
		log.Error("Formula evaluation errored",
			"err", result.Error.Message(),
		)
	} else {
		log.Info("Formula evaluation finished", log15.Ctx{
			"exitcode": result.ExitCode,
			"outputs":  result.Outputs,
		})
	}
	return nil
}
