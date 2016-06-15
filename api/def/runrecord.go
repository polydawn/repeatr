package def

import (
	"time"
)

/*
	A RunID is created at the start of an evaluation of a formula.
	It can be used to follow the evaluation's progress.

	A RunID is an arbitrary guid (there's nothing else unique to go on
	at the time a formula evaluation begins).
*/
type RunID string

type RunRecord struct {
	// Hash ID (derived; actually includes the UID; always do lookups by this, not the UID).
	HID string `json:"HID"`

	// Unique ID, arbitrarily selected.
	UID RunID `json:"UID"`

	// Date of formula execution.
	Date time.Time `json:"when"`

	// Which formula was executed.
	FormulaHID string `json:"formulaHID"`

	// Results!
	Results ResultGroup `json:"results"`

	// ... or Error, for major issues during the run.
	Failure error `json:"failure,omitempty"`
}

type ResultGroup map[string]*Result

/*
	`Result`s are produced when gathering up data as defined by an `Output` at
	the end of running a `Formula`.  It just includes the name from the formula
	and the bare ware information -- filters, mountpaths... all that stuff is in
	the past now.
*/
type Result struct {
	Name string `json:"-"`
	Ware
}
