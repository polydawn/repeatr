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

func ensureHomeDir(rootfsPath string, policy Policy) {

}

func ensureTempDir(rootfsPath string, policy Policy) {

}

func ensureIdentity(rootfsPath string, policy Policy) {

}
