package scheduler

import (
	"github.com/spacemonkeygo/errors"
)

// grouping, do not instantiate
var Error *errors.ErrorClass = errors.NewClass("SchedulerError")

// error when a scheduler's queue was full and could not enqueue a forumla.
var QueueFullError *errors.ErrorClass = Error.NewClass("QueueFullError")
