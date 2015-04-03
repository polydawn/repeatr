package output

import (
	"github.com/spacemonkeygo/errors"
)

var Error *errors.ErrorClass = errors.NewClass("OutputError") // grouping, do not instantiate

/*
	Indicates that the target filesystem (the one given to `Apply`) had some error.
*/
var TargetFilesystemUnavailableError *errors.ErrorClass = Error.NewClass("TargetFilesystemUnavailableError")

// Convenience method for wrapping io errors.
func TargetFilesystemUnavailableIOError(err error) *errors.Error {
	return TargetFilesystemUnavailableError.Wrap(errors.IOError.Wrap(err)).(*errors.Error)
}

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("OutputUnknownError")
