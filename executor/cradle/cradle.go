package cradle

import (
	"os"
	"path/filepath"

	"polydawn.net/repeatr/def"
)

/*
	Ensure the MVP filesystem, mkdir'ing and mutating as necessary.

	`ApplyDefaults` first (this refers to cwd).
*/
func MakeCradle(rootfsPath string, frm def.Formula) {

}

// TODO : also support support empty dir as an input type for freehand
// Note: does *not* ensure that the working dir is empty.
func ensureWorkingDir(rootfsPath string, frm def.Formula) {
	os.MkdirAll(filepath.Join(rootfsPath, frm.Action.Cwd), 0755) // TODO root up to tip?  or all new dirs at $uid?
}

func ensureHomeDir(rootfsPath string, policy def.Policy) {

}

func ensureTempDir(rootfsPath string, policy def.Policy) {

}

/*
	Ensures the MVP filesystem considers configuration for the appropriate
	user account.

	Which is "the appropriate user account" varies according to the Policy.

	We define identity in terms that `nsswitch` would refer to as "compat":
	the ancient `/etc/{passwd,group,shadow}` files, because these are the
	most widely understood formats, and tend to keep working even in
	non-dynamically-linked programs (and different libc implementations,
	etc etc etc).

	We do not screw with a `/etc/nsswitch.conf` file if one exists, nor
	alter our behavior based on it -- cradle is about enforcing *absolute*
	minimum viable behaviors and fallbacks, not parsing and smoothing
	every concievable fractal of at-one-time-in-history-valid configuration.

*/
func ensureIdentity(rootfsPath string, policy def.Policy) {

}
