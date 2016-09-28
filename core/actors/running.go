package actor

import (
	"bufio"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/inconshreveable/log15"

	"go.polydawn.net/repeatr/api/act"
	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/lib/iofilter"
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

func (frCfg *FormulaRunnerConfig) StreamFollower(job executor.Job, wr io.Writer, fds []int, wg *sync.WaitGroup) {
	r := iofilter.Reframer{
		Delegate:  wr,
		SplitFunc: bufio.ScanLines,
		Prefix:    []byte{},
		Suffix:    []byte{'\n'},
	}

	io.Copy(&r, job.Outputs().Reader(fds...))
	wg.Done()
}

func (frCfg *FormulaRunnerConfig) FollowStreams(runID def.RunID, stdout io.Writer, stderr io.Writer) {
	job := frCfg.wards[runID]
	var wg sync.WaitGroup
	if stdout == stderr {
		wg.Add(1)
		go frCfg.StreamFollower(job, stdout, []int{1, 2}, &wg)
	} else {
		wg.Add(2)
		go frCfg.StreamFollower(job, stdout, []int{1}, &wg)
		go frCfg.StreamFollower(job, stdout, []int{2}, &wg)
	}
	wg.Wait()
}

func (frCfg *FormulaRunnerConfig) FollowResults(runID def.RunID) *def.RunRecord {
	job := frCfg.wards[runID]
	jr := job.Wait()

	// Temporary: flip results types.  (TODO: keep driving this version deeper.)
	results := def.ResultGroup{}
	for name, output := range jr.Outputs {
		results[name] = &def.Result{name,
			def.Ware{output.Type, output.Hash},
		}
	}

	// Place the exit code among the results.
	//  This is so a caller can unambiguously see the job's exit code;
	//  while we do attempt to forward a pass-vs-fail signal through our
	//  own exit code by default, we can only piggyback so much signal;
	//  we also need space to report our own errors distinctly.
	results["$exitcode"] = &def.Result{"$exitcode",
		def.Ware{"exitcode", strconv.Itoa(jr.ExitCode)},
	}

	return &def.RunRecord{
		UID:        def.RunID(job.Id()),
		Date:       time.Now().Truncate(time.Second), // FIXME elide this translation layer, this should be committed just once
		FormulaHID: "todo",                           // FIXME write formula HID
		Results:    results,
		Failure:    jr.Error,
	}
}
