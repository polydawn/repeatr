package act

import (
	"io"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
)

/*
	The RunObserver interface exposes functions to observe the progress of
	running.

	Implementations may be realtime following local exec, or used to observe
	event streams over the network, or from logs.
*/
type RunObserver interface {
	/*
		Subscribes to following an Event stream for the given RunID.
		An offset to start from is optional (currently not implemented).

		May panic with:

		  - `*act.ErrRunIDNotFound` if this observer doesn't have that RunID.
		  - `*act.ErrRemotePanic` in the case of invalid values in the stream.
	*/
	FollowEvents(
		which def.RunID,
		stream chan<- *def.Event,
		startingFrom def.EventSeqID,
		// type filter?  may not want journal lines, for example.
	)
}

type ErrRemotePanic struct {
	meep.TraitAutodescribing
	Dump string
}

type ErrRunIDNotFound struct {
	meep.TraitAutodescribing
	RunID def.RunID
}

/*
	A RunProvider implementor can take formulas and start running them.

	Most RunProvider implementations are also a RunObserver implementation,
	because it's pretty useless to be able to launch things without being
	able to monitor them.
*/
type RunProvider interface {
	/*
		Schedule a new run, immediately returning an ID that can be used to
		follow it.
	*/
	StartRun(*def.Formula) def.RunID
}

// BELOW: DEPRECATED.

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
