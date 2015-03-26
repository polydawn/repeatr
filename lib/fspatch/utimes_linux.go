package fspatch

import (
	"os"
	"syscall"
	"unsafe"
)

func LUtimesNano(path string, ts []syscall.Timespec) error {
	// These are not currently available in syscall
	AT_FDCWD := -100
	AT_SYMLINK_NOFOLLOW := 0x100

	var _path *byte
	_path, err := syscall.BytePtrFromString(path)
	if err != nil {
		return &os.PathError{"chtimes", path, err}
	}

	// Note this does depend on kernel 2.6.22 or newer.  Fallbacks are available but we haven't implemented them and they lose nano precision.
	if _, _, err := syscall.Syscall6(syscall.SYS_UTIMENSAT, uintptr(AT_FDCWD), uintptr(unsafe.Pointer(_path)), uintptr(unsafe.Pointer(&ts[0])), uintptr(AT_SYMLINK_NOFOLLOW), 0, 0); err != 0 {
		return &os.PathError{"chtimes", path, err}
	}

	return nil
}

func UtimesNano(path string, ts []syscall.Timespec) error {
	// Note that this is disambiguated from plain `os.Chtimes` only in that it refuses to fall back to lower precision on old kernels.
	// Like LUtimesNano, it depends on kernel 2.6.22 or newer.
	if err := syscall.UtimesNano(path, ts); err != nil {
		return &os.PathError{"chtimes", path, err}
	}
	return nil
}
