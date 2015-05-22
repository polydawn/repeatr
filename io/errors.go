package integrity

import (
	"github.com/spacemonkeygo/errors"
)

var Error *errors.ErrorClass = errors.NewClass("IntegrityError") // grouping, do not instantiate

/*
	Raised to indicate that some configuration is missing or malformed.
*/
var ConfigError *errors.ErrorClass = Error.NewClass("ConfigError")

/*
	Indicates that an error getting the data from siloed storage.
	(As contrasted with a `TargetFilesystemUnavailableError`, which is
	an error that comes while trying to unpack the data source onto a
	local filesystem for use.)

	Examples include a URI that specified a file that doesn't exist, or
	an IO error later while reading from that source, etc.

*/
var DataSourceUnavailableError *errors.ErrorClass = Error.NewClass("DataSourceUnavailableError")

// Convenience method for wrapping io errors.
func DataSourceUnavailableIOError(err error) *errors.Error {
	return DataSourceUnavailableError.Wrap(errors.IOError.Wrap(err)).(*errors.Error)
}

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.

	If an error of this type is exposed to the user, it should be
	considered a bug, and specific error detection added to the site.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("IntegrityUnknownError")
