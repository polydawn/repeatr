package fspatch

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

func LUtimesNano(path string, atime time.Time, mtime time.Time) error {
	var utimes [2]syscall.Timespec
	utimes[0] = syscall.NsecToTimespec(atime.UnixNano())
	utimes[1] = syscall.NsecToTimespec(mtime.UnixNano())

	var _path *byte
	_path, err := syscall.BytePtrFromString(path)
	if err != nil {
		return err
	}

	if _, _, err := syscall.Syscall(syscall.SYS_LUTIMES, uintptr(unsafe.Pointer(_path)), uintptr(unsafe.Pointer(&utimes[0])), 0); err != 0 {
		return &os.PathError{"chtimes", path, err}
	}

	return nil
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
