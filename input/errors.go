package input

import (
	"github.com/spacemonkeygo/errors"
)

var Error *errors.ErrorClass = errors.NewClass("InputError") // grouping, do not instantiate

/*
	Indicates that the input failed to obtain data that correctly
	matches the hash specifying the input.  This may mean there have
	been data integrity issues in the storage or transport systems involved.
*/
var InputHashMismatchError *errors.ErrorClass = Error.NewClass("InputHashMismatchError")

/*
	Indicates that the target filesystem (the one given to `Apply`) had some error.
*/
var TargetFilesystemUnavailableError *errors.ErrorClass = Error.NewClass("InputTargetFilesystemUnavailableError")

// Convenience method for wrapping io errors.
func TargetFilesystemUnavailableIOError(err error) *errors.Error {
	return TargetFilesystemUnavailableError.Wrap(errors.IOError.Wrap(err)).(*errors.Error)
}

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("InputUnknownError")
