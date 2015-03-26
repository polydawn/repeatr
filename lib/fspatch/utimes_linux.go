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

	// These are not currently available in syscall
	AT_FDCWD := -100
	AT_SYMLINK_NOFOLLOW := 0x100

	var _path *byte
	_path, err := syscall.BytePtrFromString(path)
	if err != nil {
		return &os.PathError{"chtimes", path, err}
	}

	// Note this does depend on kernel 2.6.22 or newer.  Fallbacks are available but we haven't implemented them and they lose nano precision.
	if _, _, err := syscall.Syscall6(syscall.SYS_UTIMENSAT, uintptr(AT_FDCWD), uintptr(unsafe.Pointer(_path)), uintptr(unsafe.Pointer(&utimes[0])), uintptr(AT_SYMLINK_NOFOLLOW), 0, 0); err != 0 {
		return &os.PathError{"chtimes", path, err}
	}

	return nil
}

func UtimesNano(path string, atime time.Time, mtime time.Time) error {
	// Note that this is disambiguated from plain `os.Chtimes` only in that it refuses to fall back to lower precision on old kernels.
	// Like LUtimesNano, it depends on kernel 2.6.22 or newer.
	var utimes [2]syscall.Timespec
	utimes[0] = syscall.NsecToTimespec(atime.UnixNano())
	utimes[1] = syscall.NsecToTimespec(mtime.UnixNano())
	if err := syscall.UtimesNano(path, utimes[0:]); err != nil {
		return &os.PathError{"chtimes", path, err}
	}
	return nil
}
