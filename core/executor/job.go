package executor

// Note: this interface is entirely dubious.
// - streams don't necessarily work like that
// - jobID is just kind of generally spacey
// - the results struct is simply a hot mess
// ... almost all of this should be replaced by `def.RunRecord` things.
// Output *streams* are still interesting, but need to be accessible via `api/act`.

import (
	"io"

	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/lib/streamer"
)

/*
	Job is an interface for observing actively running tasks.
	It contains progress reporting interfaces, streams that get realtime
	stdout/stderr, wait for finish, return exit codes, etc.

	All of the data available from `Job` should also be accessible as
	some form of `Output`s	after the execution is complete, but `Job` can
	provide them live.
*/
type Job interface {
	// question the first: provide readables, or accept writables for stdout?
	// probably provide.  the downside of course is this often forces a byte copy somewhere.
	// however, we're going to store these streams anyway.  so the most useful thing to do actually turns out to be log outputs immediately, and just reexpose that readers to that stream.

	Id() JobID // the ID of this job

	/*
		Returns a new reader that delivers the combined stdout and
		stderr of a command, from the beginning of execution.

		Shorthand for `Outputs().Reader(1, 2)`.
	*/
	OutputReader() io.Reader

	/*
		Returns a mux of readable streams.  Numbering is as typical
		unix convention (1=stdout, 2=stderr, etc).
	*/
	Outputs() streamer.ROMux

	/*
		Waits for completion if necessary, then returns the job's results
	*/
	Wait() JobResult
}

type JobID string // type def just to make it hard to accidentally get ids crossed.

/*
	Very much a work-in-progress.

	Holds all information you might want from a completed Job.
*/
type JobResult struct {
	ID JobID

	Error *errors.Error // if the executor experienced a problem running this job. REVIEW: type discussion? semantics?

	ExitCode int // The return code of this job

	Outputs def.OutputGroup //The hashed outputs from this job

	// More?
}
