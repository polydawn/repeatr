package act

import (
	"io"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
)

/*
	The FormulaRunner interface exposes functions to start running things,
	and to observe their progress.
*/
type FormulaRunner interface {
	/*
		Queues a Formula to run, and returns the RunID by which it can
		be addressed and watched.
	*/
	StartRun(*def.Formula) def.RunID

	/*
		Subscribes to following an Event stream for the given RunID.
		An offset to start from is optional (currently not implemented).
	*/
	FollowEvents(
		which def.RunID,
		stream chan<- def.Event,
		startingFrom def.EventSeqID,
		// type filter?  may not want journal lines, for example.
	)

	/*
		Waits for the run to finish completely, and returns the RunRecord.
	*/
	AwaitRunRecord(def.RunID) *def.RunRecord
}

type ErrRemotePanic struct {
	meep.TraitAutodescribing
	Dump string
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
