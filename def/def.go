/*
	Repeatr is focused on telling a story about formulas: when you put the same
	things in, you should get the same things out.

	Formulas describe a piece of computation, its inputs, and how to collect
	its outputs.  After that, repeatr can help you make sure your formula
	produces the same thing time and time again.

	We call the parts of the formula that should be deterministic the "conjecture".
	We'll use this word consistently throughout the documentation.
	Anything that is part of the conjecture is hashed when processing the formula,
	and any output marked as part of the conjecture is hashed after the formula's task
	is executed.  (You can choose which of the outputs are part of your conjecture!
	But everything about your inputs must be part of your conjecture, because
	if the inputs change, output consistency is impossible -- except stuff like
	the network locations of data is skipped from the conjecture, since
	that can change without changing the meaning of your formula.)

	### Mathwise:

	Given a Formula j, and the []Output v, and some hash h:

	h(j.Inputs||j.Action||filter(j.Outputs, where Conjecture=true)) -> h(v)

	should be an onto relationship.

	In other words, a Formula should define a "pure" function.  And we'll let you know if it doesn't.

	### Misc docs:

	- The root filesystem of your execution engine is just another `Input` with the rest, with MountPath="/".
	Exactly one input with the root location is required at runtime.

	- Formula.SchedulingInfo, since it's *not* included in the 'conjecture',
	is expected not to have a major impact on your execution correctness.
*/
package def

import (
	"io"
	"time"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/lib/streamer"
)

/*
	Formula describes `(inputs, computation) -> (outputs)`.

	Values may be mutated during final validation if missing,
	i.e. the special `Output` that describes stdout and stderr is required
	and will be supplied for you if not already specifically configured.
*/
type Formula struct {
	Inputs  []Input  `json:"inputs"`  // total set of inputs.  sorted order.  included in the conjecture.
	Action  Action   `json:"action"`  // description of the computation to be performed.  included in the conjecture.
	Outputs []Output `json:"outputs"` // set of expected outputs.  sorted order.  conditionally included in the conjecture (configurable per output).
	//SchedulingInfo interface{} // configures what execution framework is used and impl-specific additional parameters to that (minimum node memory, etc).  not considered part of the conjecture.
}

/*
	Input specifies a data source to feed into the beginning of a computation.

	Inputs can be one of many different `Type`s of data source.
	Examples may include "tar", "git", "hadoop", "ipfs", etc.

	Inputs must specify both a `Hash` and a `URI`.
	`Input.Hash` is the content identity descriptor and will always be verified for all types of data source.
	`Input.Hash` is both identifies the data and provides integrity (in other words,
	all repeatr's input types will use a cryptographically strong hash here,
	so given a hash even an untrusted datastore is safe to use).
	Repeatr requires this to be accurate because if the inputs change, output
	consistency is impossible -- so even for plain http downloads, this is enforced.

	`Input.URI` is a secondary content lookup descriptor, like where on
	the filesystem or network information can be fetched from.
	`Input.URI` might contain extra description to answer questions like
	"which git remote url should i fetch from" or
	"which ipfs swarm do i coordinate with".

	The `URI` is *not* included in the conjecture, because repeatr understands
	that your data will be mobile -- that's why we have the `Input.Hash` take the leading role
	and why the `Input.Hash` should be sufficient to identify the information.
	Changes in the `Input.URI` field may make or break whether the data is accessible,
	but should never actually change the content of the data -- it's just supposed to talk about
	transport details; and content itself is still checked by `Input.Hash`.
*/
type Input struct {
	Name      string // the map key.  by default will also be used for MountPath.
	Type      string `json:"type"`  // implementation name (repeatr-internal).  included in the conjecture.
	Hash      string `json:"hash"`  // identifying hash of input data.  included in the conjecture.
	URI       string `json:"silo"`  // secondary content lookup descriptor.  not considered part of the conjecture.
	MountPath string `json:"mount"` // filepath where this input should be mounted in the execution context.  included in the conjecture.
}

type InputsByName []Input

func (a InputsByName) Len() int           { return len(a) }
func (a InputsByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a InputsByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

/*
	Action describes the computation to be run once the inputs have been set up.
	All content is part of the conjecture.
*/
type Action struct {
	Entrypoint []string          `json:"command"` // executable to invoke as the task.  included in the conjecture.
	Cwd        string            `json:"cwd"`     // working directory to set when invoking the executable.  if not set, will be defaulted to "/".
	Env        map[string]string `json:"env"`     // environment variables.  included in the conjecture.
}

/*
	Output describes where we intend to pick up data after a task completes.

	Outputs can be one of many different `Type`s of data sink.
	Examples may include "tar", "git", "hadoop", "ipfs", etc.

	`Output.MountPath` states where we should collect information from the
	task execution environment.
	After the task completes, repeatr will pick up this data, ship it off
	to storage, and also calculate a checksum of the data so we can see
	whether it matches any prior (or future) runs of this `Formula`.

	Outputs must specify a `URI`; repeatr will ship your data to this address.
	`Output.URI` has similar properties to `Input.URI` (and also similarly,
	is not included in the conjecture, because repeatr understands that
	your data can be mobile).

	The `Output.Hash` field will be filled in with a value computed
	from the data present in `Output.MountPath` after the task has completed.
	As with `Input.Hash`, the `Output.Hash` in repeatr will always be a
	cryptographically strong hash, which means it precisely describes your
	data, and makes it virtually impossible to accidentally get the same
	`Hash` as other data -- any changes to your output will always result
	in a very different `Hash` value.

	(In a content-addressable data store, repeatr may just lift the data store's
	address to use as `Output.Hash`, which is super efficient for everyone involved.
	For other more legacy-oriented systems, this may be a hash of the
	of the working filesystem right before before export.)

	Whether or not to include an `Output` in the overall `Formula`'s conjecture
	is up to you!  Many things in the world are not deterministic; repeatr
	is here to help you with the ones that should be, and stay out of the way
	for the ones that aren't.  Just set the `Output.Conjecture` boolean.

	Some examples of using `Conjecture` conditionally: if you have a job
	which does a bunch of calculations and should spit out a consistent result,
	but also does a lot of progress logging, gather those in two separate outputs.
	Mark the output of your computation in one output and set that to be
	included in the conjecture so repeatr can help you check your algorithm's
	correctness.  Now, since you may want to keep your logs for later, mark
	those as another output, and since these probably contain timestamps and
	other info that isn't *supposed* to be repeated exactly on another run,
	just set `Conjecture=false` on this one so repeatr knows not to check.

	`Output.Filters` may also be used to do a clean up pass on output files
	before committing them to storage or doing repeatr's consistency checks.
	(One typical example, which is engaged by default for you when an output
	is configured to be included in the conjecture, is setting all the file
	modification times to a standard value.)
*/
type Output struct {
	Name       string  // the map key.  by default will also be used for MountPath.
	Type       string  `json:"type"`           // implementation name (repeatr-internal).  included in the conjecture (iff the whole output is).
	Hash       string  `json:"hash"`           // identifying hash of output data.  generated by the output handling implementation during data export when a task is complete.  included in the conjecture (iff the whole output is).
	URI        string  `json:"silo,omitempty"` // where to ship the output data.  not considered part of the conjecture.
	MountPath  string  `json:"mount"`          // filepath where this output will be yanked from the job when it reaches completion.  included in the conjecture (iff the whole output is).
	Filters    Filters `json:"filters,omitempty"`
	Conjecture bool    `json:"cnj,omitempty"` // whether or not this output is expected to contain the same result, every time, when given the same set of `Input` items.
}

type OutputsByName []Output

func (a OutputsByName) Len() int           { return len(a) }
func (a OutputsByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a OutputsByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

/*
	Filters are transformations that can be applied to data, either to
	normalize it for storage or to apply attributes to it before feeding
	the data into an action's inputs.

	The following filters are available:

		- uid   -- the posix user ownership number
		- gid   -- the posix group ownership number
		- mtime -- the posix file modification timestamp

	'uid', 'gid', and 'mtime' are all filtered by default on formula outputs --
	most use cases do not need these attributes, and they are a source of nondeterminism.
	If you want to keep them, you may specify	`uid keep`, `gid keep`, `mtime keep`,
	etc; if you want the filters to flatten to different values than the defaults,
	you may specify `uid 12000`, etc.
	(Note that the default mtime filter flattens the time to Jan 1, 2010 --
	*not* epoch.  Some contemporary software has been known to regard zero/epoch
	timestamps as errors or empty values, so we've choosen a different value in
	the interest of practicality.)

	Filters on inputs will be applied after the data is fetched and before your
	job starts.
	Filters on outputs will be applied after your job process exits, but before
	the output hash is computed and the data committed to any warehouses for storage.

	Note that these filters are built-ins (and there are no extensions possible).
	If you need more complex data transformations, incorporate it into your job
	itself!  These filters are built-in because they cover the most common sources
	of nondeterminism, and because they are efficient to implement as special
	cases in the IO engines (and in some cases, e.g. ownership filters, are also
	necessary for security properties an dusing repeatr IO with minimal host
	system priviledges).
*/
type Filters struct {
	UidMode   FilterMode
	Uid       int
	GidMode   FilterMode
	Gid       int
	MtimeMode FilterMode
	Mtime     time.Time
}

type FilterMode int

const (
	FilterUninitialized FilterMode = iota
	FilterUse
	FilterKeep
	FilterHost
)

var (
	FilterDefaultUid   = 1000
	FilterDefaultGid   = 1000
	FilterDefaultMtime = time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)
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

	Outputs []Output //The hashed outputs from this job

	// More?
}
