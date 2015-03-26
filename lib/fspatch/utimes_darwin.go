package fspatch

import (
	"os"
	"syscall"

	"polydawn.net/repeatr/def"
)

func LUtimesNano(path string, ts []syscall.Timespec) error {
	return def.ErrNotSupportedPlatform
}

func UtimesNano(path string, ts []syscall.Timespec) error {
	if err := syscall.UtimesNano(path, ts); err != nil {
		return &os.PathError{"chtimes", path, err}
	}
	return nil
}
