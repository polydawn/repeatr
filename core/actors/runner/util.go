package runner

import (
	"strconv"
	"time"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
)

// Bridge method for executor.Job to def.RunRecord.
// May be a refactor target; can remove if executor just uses RunRecord.
// (There *is* a long standing comment line in job.go about "almost all of this should be replaced by `def.RunRecord` things" already...)
func jobToRunRecord(job executor.Job) *def.RunRecord {
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
		UID:     def.RunID(job.Id()),
		Date:    time.Now().Truncate(time.Second), // FIXME elide this translation layer, this should be committed just once
		Results: results,
		Failure: jr.Error,
	}
}
