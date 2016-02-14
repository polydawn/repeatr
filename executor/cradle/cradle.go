package cradle

import (
	"polydawn.net/repeatr/def"
)

func MakeCradle(rootfsPath string, policy Policy) {

}

// TODO : also support support empty dir as an input type for freehand
// Note: does *not* ensure that the working dir is empty.
func ensureWorkingDir(rootfsPath string, policy Policy) {

}

func ensureHomeDir(rootfsPath string, policy Policy) {

}

func ensureTempDir(rootfsPath string, policy Policy) {

}

func ensureIdentity(rootfsPath string, policy Policy) {

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
