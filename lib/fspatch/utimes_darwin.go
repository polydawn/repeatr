package fspatch

import (
	"os"
	"syscall"
	"time"

	"go.polydawn.net/repeatr/api/def"
)

func LUtimesNano(path string, atime time.Time, mtime time.Time) error {
	return def.ErrNotSupportedPlatform
}

func UtimesNano(path string, atime time.Time, mtime time.Time) error {
	var utimes [2]syscall.Timespec
	utimes[0] = syscall.NsecToTimespec(atime.UnixNano())
	utimes[1] = syscall.NsecToTimespec(mtime.UnixNano())
	if err := syscall.UtimesNano(path, utimes[0:]); err != nil {
		return &os.PathError{"chtimes", path, err}
	}
	return nil
}
