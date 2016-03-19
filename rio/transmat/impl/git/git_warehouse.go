package git

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/rio"
)

/*
	Refer to, interact with, and manage a warehouse.

	Since this is git we're talking about, this is basically references
	to another git repo.
*/
type Warehouse struct {
	url string
}

/*
	Initialize a warehouse controller.

	Note: parsing git URLs is *hard*.  This function is "best effort"
	stuff and may not be able to error in all circumstances where the
	git commands will error.  See `man gitremote-helpers` for one reason
	this is the edge of a tarpit.  (This may improve as we do a better job
	of pinning specific git versions and sandboxing their environment,
	but at the moment, caveat emptor, and this is "PRs welcome" turf.)

	May panic with:
	  - Config Error: if the URI is unparsable or has an unsupported scheme.
*/
func NewWarehouse(coords rio.SiloURI) *Warehouse {
	wh := &Warehouse{}
	wh.url = hammerRelativePaths(string(coords))
	return wh
}

/*
	Desperately attempt to sanitize paths for git.

	This is almost certainly flawed; but frankly, so is git's handling of
	these things: it's not consistent across all commands (specifically,
	`git clone` and `git ls-remote`).  We're going for best-effort here,
	and slightly-better-than-not-trying -- low bars though those are.

	Git has a MAJORLY hard time consistently understanding local paths; so,
	we give up; internally all paths shall be absolutized because there's
	simply no functional choice.

	Here's a short list of issues we're concerned with:

	  - There's no "--" in ls-remote, so in order to avoid ambiguity with
	    arguments, we forbid things starting in "-".
	  - `git ls-remote .` and `git ls-remote ./` are a special case and will
	    return results even if the repo root is *above* your cwd, even
	    though git clone will disagree for obvious reasons.
	  - `git ls-remote ../sibling` works fine (and so does clone), but if
	    you do it from *inside* the same repo, it just says no-such-repo
	    (...but clone still works!)
	  - `ls-remote` in particular is really willing to take you on a ride:
	    if you have a repo-A, you make another git repo-B *inside* repo-A,
	    then you can `ls-remote ..` correctly....
	  - ....but if you then create a dir inside of repo-B, cd into it, and
	    try `git ls-remote ../` again you'll get the info from **repo-A**,
	    the grandparent.
	  - So basically, relative paths in git simply Don't Work and are
	    apparently Not Supported.

	Beyond that, we mostly punt.  It's essentially impossible to validate
	everything in advance in the same way that git will feel.
	The area we're *really* minding here is where we've observed
	`git clone` and `git ls-remote` behaviors to drift apart, because
	that's exceptionally nasty to deal with or report about clearly.
*/
func hammerRelativePaths(coords string) string {
	if len(coords) < 1 {
		return coords
	}
	// If things start with a "-", just... no.  Git lacks consistent ability
	//  to handle this unambiguously, so we're not even going to try.
	if coords[0] == '-' {
		panic(rio.ConfigError.New("invalid git remote: cannot start with '-'"))
	}
	// If something looks like a relative path, absolutize it.
	//  There's a *litany* of issues this works around.
	//  If you had an ssh URL with a username starting in dot, eh, god help you.
	//  Note that we're not chasing after "file://" stuff here; git has
	//   even more other opinions about that already.
	if coords[0] == '.' {
		abs, err := filepath.Abs(coords)
		if err != nil {
			panic(rio.TransmatError.Wrap(err))
		}
		return abs
	}
	// There's no point in trying to parse the rest as a URL and sanitize it;
	//  it's both practically and theoretically impossible to accurately seek
	//   parity with how git may choose to see things.
	return coords
}

/*
	Check if the warehouse exists and can be contacted.

	Returns nil if contactable; if an error, the message will be
	an end-user-meaningful description of why the warehouse is out of reach.
*/
func (wh *Warehouse) Ping() *errors.Error {
	// Shell out to git and ask it if it thinks there's a repo here.
	//  `git ls-remote` is our best option here for checking out that location and making sure it's advertising refs,
	//    while refraining from any alarmingly heavyweight operations or data transfers.
	var errBuf bytes.Buffer
	code := git.Bake(
		"ls-remote", wh.url,
		gosh.Opts{
			// never buffer stdout; it may be long and we don't care.
			Err:    &errBuf,
			OkExit: gosh.AnyExit,
		},
	).Run().GetExitCode()
	switch code {
	case 0:
		return nil
	case 128:
		// Code 128 appears to result from any cant-fetch scenario.
		// So far, we've also only seen error messages where the first line
		//  is interesting, so that's what we report.
		msg := strings.TrimPrefix(strings.SplitN(errBuf.String(), "\n", 2)[0], "fatal: ")
		// Known values include:
		//  - "'%s' does not appear to be a git repository"
		//  - "attempt to fetch/clone from a shallow repository"
		return rio.WarehouseUnavailableError.New("git remote unavailable: %s", msg).(*errors.Error)
	default:
		// We don't recognize this.
		panic(rio.UnknownError.New("git exit code %d (stderr: %s)", code, errBuf.String()))
	}
}
