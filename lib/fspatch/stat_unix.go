// +build linux darwin
// can probably be built on most of the platforms listed in tar/archive/stat_unix.go, but not sure so preferring to whitelist cautiously

package fspatch

import (
	"os"
	"syscall"
)

/*
	Get the device numbers from a fileinfo.
	Only makes sense on block and character devices.
*/
func ReadDev(fi os.FileInfo) (devmajor, devminor int64) {
	sys, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	devmajor = int64((sys.Rdev >> 8) & 0xff)
	devminor = int64((sys.Rdev & 0xff) | ((sys.Rdev >> 12) & 0xfff00))
	return
}
