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
	ensureWorkingDir(rootfsPath, frm)
	ensureHomeDir(rootfsPath, frm.Action.Policy)
	ensureTempDir(rootfsPath)
	ensureIdentity(rootfsPath, frm.Action.Policy)
}

// TODO : also support support empty dir as an input type for freehand
// Note: does *not* ensure that the working dir is empty.
func ensureWorkingDir(rootfsPath string, frm def.Formula) {
	os.MkdirAll(filepath.Join(rootfsPath, frm.Action.Cwd), 0755) // TODO root up to tip?  or all new dirs at $uid?
}

func ensureHomeDir(rootfsPath string, policy def.Policy) {

}

/*
	Ensure `/tmp` exists and anyone can write there.
	The sticky bit will be applied and permissions set to 777.

	Edge case note: will follow symlinks.
*/
func ensureTempDir(rootfsPath string) {
	pth := filepath.Join(rootfsPath, "/tmp")
	stickyMode := os.FileMode(0777) | os.ModeSticky
	// try to chmod first
	err := os.Chmod(pth, stickyMode)
	if err == nil {
		return
	}
	// for unexpected complaints, bail
	if !os.IsNotExist(err) {
		panic(err) // TODO category for here??
	}
	// mkdir if not exist
	if err := os.Mkdir(pth, stickyMode); err != nil {
		panic(err) // TODO category for here??
	}
	// chmod it *again* because unit tests reveal that `os.Mkdir` is subject to umask
	if err := os.Chmod(pth, stickyMode); err != nil {
		panic(err) // TODO category for here??
	}
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
