package def

import (
	"github.com/spacemonkeygo/errors"
)

/*
	Validation error is a base class for anything that matches the description
	of an HTTP 400.  (Unless the validation should have been performed at an
	earlier stage, and the current check is only for sanity; then, if it fails
	and it's considered a compile-time boo boo, use `errors.ProgrammerError`.)
*/
var ValidationError *errors.ErrorClass = errors.NewClass("ValidationError")
