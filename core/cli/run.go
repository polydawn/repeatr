package cli

import (
	"io"

	"github.com/inconshreveable/log15"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/actors"
	"go.polydawn.net/repeatr/core/executor"
)

func RunFormula(execr executor.Executor, formula def.Formula, output io.Writer, journal io.Writer, serialize bool) *def.RunRecord {
	log := log15.New()

	if serialize {
		// use our custom logHandler to serialize results uniformly
		log.SetHandler(logHandler(output))
	} else {
		// no serialization of output, write directly to journal
		log.SetHandler(log15.StreamHandler(journal, log15.TerminalFormat()))
	}

	// Create a local formula runner, and fire.
	runner := actor.NewFormulaRunner(execr, log)
	runID := runner.StartRun(&formula)

	if serialize {
		// set up serializer for journal stream
		js := &journalSerializer{
			Delegate: output,
			RunID:    runID,
		}
		journal = js
	}

	// Stream job output to terminal in real time
	//  (stderr and stdout of the job both go to the same stream as our own logs).
	runner.FollowStreams(runID, journal, journal)

	// Wait for results.
	result := runner.FollowResults(runID)
	if result.Failure == nil {
		log.Info("Job finished",
			"outputs", result.Results,
		)
	} else {
		log.Error("Job execution errored",
			"error", result.Failure,
		)
	}
	return result
}
