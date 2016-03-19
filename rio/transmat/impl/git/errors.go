package git

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/rio"
)

/*
	Catch-all error for git subprocesses.

	Git is a fractal of error handling and fuzzy string matching, so this
	is used more than one might like.
*/
var Error *errors.ErrorClass = rio.Error.NewClass("GitError")
