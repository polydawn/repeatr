package fspatch

import (
	"os"
	"syscall"
	"unsafe"
)

func LUtimesNano(path string, ts []syscall.Timespec) error {
	var _path *byte
	_path, err := syscall.BytePtrFromString(path)
	if err != nil {
		return &os.PathError{"chtimes", path, err}
	}

	if _, _, err := syscall.Syscall(syscall.SYS_LUTIMES, uintptr(unsafe.Pointer(_path)), uintptr(unsafe.Pointer(&ts[0])), 0); err != 0 {
		return &os.PathError{"chtimes", path, err}
	}

	return nil
}

func UtimesNano(path string, ts []syscall.Timespec) error {
	if err := syscall.UtimesNano(path, ts); err != nil {
		return &os.PathError{"chtimes", path, err}
	}
	return nil
}
