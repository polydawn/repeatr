package act

import (
	"io"

	"go.polydawn.net/repeatr/api/def"
)

/*
	Schedule a new run, immediately returning an ID that can be used to
	follow it.
*/
type StartRun func(*def.Formula) def.RunID

/*
	Follow the stdout and stderr of a run.
	Starts at the beginning, and returns when both streams have been closed.
*/
type FollowStreams func(def.RunID, io.Writer, io.Writer)

/*
	Wait for the completion of a run and return its results.
*/
type FollowResults func(def.RunID) *def.RunRecord
