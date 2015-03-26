package fs

import (
	"github.com/spacemonkeygo/errors"
)

var Error *errors.ErrorClass = errors.NewClass("FSError") // grouping, do not instantiate

/*
	`BreakoutError` is raised when processing a filesystem description where links
	(symlinks, hardlinks) are constructed in such a way that they would reach
	out of the base directory.  Encountering these in a well-formed filesystem
	description is basically an attempted attack.
*/
var BreakoutError *errors.ErrorClass = Error.NewClass("FSBreakoutError")

func ioError(err error) {
	panic(Error.Wrap(errors.IOError.Wrap(err)))
}
