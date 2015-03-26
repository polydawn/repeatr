package input

import (
	"github.com/spacemonkeygo/errors"
)

type Input interface {
	/*
		Set the contents of the given filesystem path to the contents
		described by the input instance.

		Since this may be a relatively long-running operation, this method
		may return immediately, and provide a channel which may be watched
		for completion.  The channel may return an error, and must be closed
		when the filesystem manipulations are terminated.

		In the bigger picture, Executors will typically provide this this function
		with a path that's about to be mounted into a container/vm,
		or, a staging area path which is then provided to the execution
		environment via e.g. a bind mount.

		The Input implementation is allowed to assume no other process will
		mutate the filesystem (i.e. it is the executor's job to make sure
		this path is immutable by the process it is hosting).

		The Input implementation is responsible for ensuring that the content
		of this filesystem matches the hash described by the `def.Input` used
		to construct the Input implementation.

		The input expects the parent folder of its path to exist, but not the path itself.
	*/
	Apply(path string) <-chan error
}

var InputError *errors.ErrorClass = errors.NewClass("InputError") // grouping, do not instantiate

var InputHashMismatchError *errors.ErrorClass = InputError.NewClass("InputHashMismatchError")
