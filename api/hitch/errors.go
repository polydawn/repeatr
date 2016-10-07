package hitch

import (
	"go.polydawn.net/meep"
)

/*
	Error raised when deserializing just doesn't understand what it's looking
	at in the slightest; usually indicates a malformed file (or aiming at
	the wrong file, etc).
*/
type ErrParsing struct {
	meep.TraitAutodescribing
	meep.TraitCausable
	ReadingFrom string
}

/*
	Error raised for IO errors consuming input or writing output streams.
*/
type ErrIO struct {
	meep.TraitAutodescribing
	meep.TraitCausable
}
