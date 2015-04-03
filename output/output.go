package output

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
)

type Output interface {
	// See docs for input/input.go ; this is very very presumptory ATM and is liable to violent change.
	Apply(rootPath string) <-chan Report
}

type Report struct {
	Err    *errors.Error // error, or nil if success.  All errors will be under `output.Error`'s type.
	Output def.Output    // this comes back with the Hash field set
}

var Error *errors.ErrorClass = errors.NewClass("OutputError") // grouping, do not instantiate

/*
	Indicates that the target filesystem (the one given to `Apply`) had some error.
*/
var TargetFilesystemUnavailableError *errors.ErrorClass = Error.NewClass("TargetFilesystemUnavailableError")

// Convenience method for wrapping io errors.
func TargetFilesystemUnavailableIOError(err error) *errors.Error {
	return TargetFilesystemUnavailableError.Wrap(errors.IOError.Wrap(err)).(*errors.Error)
}

// wraps any other unknown errors just to emphasize the system that raised them; any well known errors should use a different type.
var UnknownError *errors.ErrorClass = Error.NewClass("OutputUnknownError")
