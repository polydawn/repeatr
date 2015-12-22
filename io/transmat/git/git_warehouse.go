package git

import (
	"bytes"
	"strings"

	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/io"
)

/*
	Refer to, interact with, and manage a warsehouse.

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
func NewWarehouse(coords integrity.SiloURI) *Warehouse {
	wh := &Warehouse{}
	// TODO we currently don't parse the URL at all, actually.
	// `url.Parse` could be made to apply, but there's really nothing we
	//  can explicitly blacklist, and we also don't internally need to
	//   do any mode-switches here (git is already and always CAS).
	wh.url = string(coords)
	return wh
}

/*
	Check if the warehouse exists and can be contacted.

	Returns nil if contactable; if an error, the message will be
	an end-user-meaningful description of why the warehouse is out of reach.
*/
func (wh *Warehouse) Ping() *errors.Error {
	// Shell out to git and ask it if it thinks there's a repo here.
	// TODO this and all future shellouts does NOT SUFFICIENTLY ISOLATE either config or secret keeping yet.
	// TODO there's no "--" in ls-remote, so... we should forbid things starting in "-", i guess?
	//  or use "file://" religiously?  but no, bc ssh doesn't look like "ssh://" all the time... ugh, i do not want to write a git url parser
	//   update: yeah, using "file://" religiously is not an option.  this actually takes a *different* path than `/non/protocol/prefixed`.  not significantly, but it may impact e.g. hardlinking, iiuc

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
		return integrity.WarehouseUnavailableError.New("git remote unavailable: %s", msg).(*errors.Error)
	default:
		// We don't recognize this.
		panic(integrity.UnknownError.New("git exit code %d (stderr: %s)", code, errBuf.String()))
	}
}
