package def

import (
	"time"
)

type RunRecord struct {
	// Hash ID (derived; actually includes the UID; always do lookups by this, not the UID).
	HID string `json:"HID"`

	// Unique ID, arbitrarily selected.
	UID string `json:"UID"`

	// Date of formula execution.
	Date time.Time `json:"when"`

	// Which formula was executed.
	FormulaHID string `json:"formulaHID"`

	// Results!
	Results ResultGroup `json:"results"`
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
