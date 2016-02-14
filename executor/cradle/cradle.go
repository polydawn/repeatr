package cradle

import (
	"polydawn.net/repeatr/def"
)

func MakeCradle(rootfsPath string, frm def.Formula) {

}

// TODO : also support support empty dir as an input type for freehand
// Note: does *not* ensure that the working dir is empty.
func ensureWorkingDir(rootfsPath string, frm def.Formula) {

}

func ensureHomeDir(rootfsPath string, frm def.Formula) {

}

func ensureTempDir(rootfsPath string, frm def.Formula) {

}

func ensureIdentity(rootfsPath string, frm def.Formula) {

}

func Env() def.Env {
	return def.Env{
		"PATH": "",
		"HOME": "",
	}
}

func DefaultCwd() string {
	return "/task"
}
