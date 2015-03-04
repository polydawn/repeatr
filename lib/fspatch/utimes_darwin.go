package fspatch

import (
	"syscall"

	"polydawn.net/repeatr/def"
)

func LUtimesNano(path string, ts []syscall.Timespec) error {
	return def.ErrNotSupportedPlatform
}

func UtimesNano(path string, ts []syscall.Timespec) error {
	return syscall.UtimesNano(path, ts)
}
