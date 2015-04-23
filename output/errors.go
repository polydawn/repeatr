package output

import (
	"github.com/spacemonkeygo/errors"
)

var Error *errors.ErrorClass = errors.NewClass("OutputError") // grouping, do not instantiate

/*
	Raised to indicate that some configuration is missing or malformed.
*/
var ConfigError *errors.ErrorClass = Error.NewClass("OutputConfigError")

/*
	Indicates that the target filesystem (the one given to `Apply`) had some error.
*/
var TargetFilesystemUnavailableError *errors.ErrorClass = Error.NewClass("OutputTargetFilesystemUnavailableError")

// Convenience method for wrapping io errors.
func TargetFilesystemUnavailableIOError(err error) *errors.Error {
	return TargetFilesystemUnavailableError.Wrap(errors.IOError.Wrap(err)).(*errors.Error)
}

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.

	If an error of this type is exposed to the user, it should be
	considered a bug, and specific error detection added to the site.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("OutputUnknownError")
