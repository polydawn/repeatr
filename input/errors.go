package input

import (
	"github.com/spacemonkeygo/errors"
)

var Error *errors.ErrorClass = errors.NewClass("InputError") // grouping, do not instantiate

/*
	Indicates that an error getting the data an input described.
	(As contrasted with a `TargetFilesystemUnavailableError`, which is
	an error that comes while trying to unpack the data source onto a
	local filesystem for use.)

	Examples include a URI that specified a file that doesn't exist, or
	an IO error later while reading from that source, etc.

*/
var DataSourceUnavailableError *errors.ErrorClass = Error.NewClass("InputDataSourceUnavailableError")

// Convenience method for wrapping io errors.
func DataSourceUnavailableIOError(err error) *errors.Error {
	return DataSourceUnavailableError.Wrap(errors.IOError.Wrap(err)).(*errors.Error)
}

/*
	Indicates that the input failed to obtain data that correctly
	matches the hash specifying the input.  This means there have
	been data integrity issues in the storage or transport systems involved --
	either the transport is having reliability issues, or, this may be an
	active attack (i.e. MITM).
*/
var InputHashMismatchError *errors.ErrorClass = DataSourceUnavailableError.NewClass("InputHashMismatchError")

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

	If an error of this type is exposed to the user, it should be
	considered a bug, and specific error detection added to the site.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("InputUnknownError")
