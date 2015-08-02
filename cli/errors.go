package cli

import (
	"github.com/spacemonkeygo/errors"
)

type ExitCode byte

const (
	EXIT_BADARGS      = ExitCode(1)
	EXIT_UNKNOWNPANIC = ExitCode(2)  // same code as golang uses when the process dies naturally on an unhandled panic.
	EXIT_USER         = ExitCode(3)  // grab bag for general user input errors (try to make a more specific code if possible/useful)
	EXIT_JOB          = ExitCode(10) // used to indicate a job reported a nonzero exit code (from cli commands that execute a single job).
)

var ExitCodeKey = errors.GenSym()

/*
	CLI errors are the last line: they should be formatted to be user-facing.
	The main method will convert a CLIError into a short and well-formatted
	message, and will *not* include stack traces unless the user is running
	with debug mode enabled.

	CLI errors are an appropriate wrapping for anything where we can map a
	problem onto something the user can understand and fix.  Errors that are
	a repeatr bug or unknown territory should *not* be mapped into a CLIError.
*/
var Error *errors.ErrorClass = errors.NewClass("CLIError")

/*
	Use this to set a specific error code the process should exit with
	when producing a `cli.Error`.

	Example: `cli.Error.New("something terrible!", SetExitCode(EXIT_BADARGS))`
*/
func SetExitCode(code ExitCode) errors.ErrorOption {
	return errors.SetData(ExitCodeKey, code)
}
