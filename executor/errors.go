package executor

import (
	"github.com/spacemonkeygo/errors"
)

// grouping, do not instantiate
var Error *errors.ErrorClass = errors.NewClass("ExecutorError")

/*
	Error raised when there are serious issues with task launch.

	Occurance of TaskExecError may be due to OS-imposed resource limits
	or other unexpected problems.  They should not be seen in normal,
	healthy operation.
*/
var TaskExecError *errors.ErrorClass = Error.NewClass("ExecutorTaskExecError")

/*
	Error raised when a command is not found inside the execution environment.

	Often just indicative of user misconfiguration (and thus this is not a
	child of TaskExecError, which expresses serious system failures).
*/
var NoSuchCommandError *errors.ErrorClass = Error.NewClass("NoSuchCommandError")

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("ExecutorUnknownError")
