package executor

import (
	"github.com/spacemonkeygo/errors"
)

// grouping, do not instantiate
var Error *errors.ErrorClass = errors.NewClass("ExecutorError")

/*
	Error raised when an executor cannot operate due to invalid setup.
*/
var ConfigError *errors.ErrorClass = Error.NewClass("ExecutorConfigError")

/*
	Error raised when there are serious issues with task launch.

	Occurance of TaskExecError may be due to OS-imposed resource limits
	or other unexpected problems.  They should not be seen in normal,
	healthy operation.
*/
var TaskExecError *errors.ErrorClass = Error.NewClass("ExecutorTaskExecError")

/*
	Error raised when job launched failed because the command is
	not found inside the execution environment.

	This is considered a form of config error since the command and
	the filesystem are both configured together, meaning a mismatch
	between them is operator error.
*/
var NoSuchCommandError *errors.ErrorClass = ConfigError.NewClass("NoSuchCommandError")

/*
	Error raised when job launched failed because the requested "cwd"
	is either not found or not a directory inside the execution environment.

	This is considered a form of config error since the command and
	the filesystem are both configured together, meaning a mismatch
	between them is operator error.
*/
var NoSuchCwdError *errors.ErrorClass = ConfigError.NewClass("NoSuchCwdError")

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.

	If an error of this type is exposed to the user, it should be
	considered a bug, and specific error detection added to the site.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("ExecutorUnknownError")
