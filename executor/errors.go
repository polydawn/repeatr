package executor

import (
	"github.com/spacemonkeygo/errors"
)

// grouping, do not instantiate
var Error *errors.ErrorClass = errors.NewClass("ChrootExecutorError")

// wraps any other unknown errors just to emphasize the system that raised them; any well known errors should use a different type.
var UnknownError *errors.ErrorClass = Error.NewClass("ChrootExecutorUnknownError")

// errors relating to task launch
// REVIEW: probably more general to executors, should be one package up and maybe wrapped with chroot.Error to express origin system (or, maybe not even, much like we haven't decided if input&output errors get wrapped)
var TaskExecError *errors.ErrorClass = Error.NewClass("ExecutorTaskExecError")

// error when a command is not found.  generally indicative of user misconfiguration (and thus not a child of TaskExecError, which expresses serious system failures).
var NoSuchCommandError *errors.ErrorClass = Error.NewClass("NoSuchCommandError")
