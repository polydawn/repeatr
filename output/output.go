package output

import (
	"github.com/spacemonkeygo/errors"
)

type Output interface {
	// stub

	// See docs for input/input.go ; this is very very presumptory ATM and is liable to violent change.
	Apply(rootPath string) <-chan error
}

var Error *errors.ErrorClass = errors.NewClass("OutputError") // grouping, do not instantiate

// wraps any other unknown errors just to emphasize the system that raised them; any well known errors should use a different type.
var UnknownError *errors.ErrorClass = Error.NewClass("OutputUnknownError")
