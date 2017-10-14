package mixins

import (
	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/rio/fs"
)

/*
	Return an error if any part of the filesystem is invalid for running the
	formula -- e.g. the CWD setting isn't a dir; the command binary
	does not exist or is not executable; etc.
	Any errors returned will be of category `ErrJobInvalid`.

	The formula is already expected to have been syntactically validated --
	e.g. all paths have been checked to be absolute, etc.  This method will
	panic if such invarients aren't held.

	(It's better to check all these things before attempting to launch
	containment because the error codes returned by kernel exec are often
	ambiguous, or provide outright misdirection with their names.
	(For example: EACCES has no less than *four* different meanings.)
	It's better that we try to detect common errors early and thus
	be able to returning meaningful and useful error messages.)

	Currently, we require exec paths to be absolute.
*/
func CheckFSReadyForExec(frm api.Formula, chrootFs fs.FS) error {
	// Check that the CWD exists and is a directory.
	stat, err := chrootFs.Stat(fs.MustAbsolutePath(string(frm.Action.Cwd)).CoerceRelative())
	if err != nil {
		return Errorf(repeatr.ErrJobInvalid, "cwd invalid: %s", err)
	}
	if stat.Type != fs.Type_Dir {
		return Errorf(repeatr.ErrJobInvalid, "cwd invalid: path is a %s, must be dir", stat.Type)
	}

	// Check that the command exists and is executable.
	//  (If the format is not executable, that's another ball of wax, and
	//  not so simple to detect, so we don't.)
	stat, err = chrootFs.Stat(fs.MustAbsolutePath(frm.Action.Exec[0]).CoerceRelative())
	if err != nil {
		return Errorf(repeatr.ErrJobInvalid, "exec invalid: %s", err)
	}
	if stat.Type != fs.Type_File {
		return Errorf(repeatr.ErrJobInvalid, "exec invalid: path is a %s, must be executable file", stat.Type)
	}
	// FUTURE: ideally we could also check if the file is properly executable,
	//  and all parents have bits to be traversable (!), to the policy uid.
	//  But this is also a loooot of work: and a correct answer (for groups
	//  at least) requires *understanding the container's groups settings*,
	//  and now you're in real hot water: parsing /etc files and hoping
	//  nobody expects nsswitch to be too interesting.  Yeah.  Nuh uh.
	//  (All of these are edge conditions tools like docker Don't Have because
	//  they simply launch you with so much privilege that it doesn't matter.)

	return nil
}
