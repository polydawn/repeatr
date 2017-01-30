// +build linux

package bind

import (
	"fmt"
	"os"
	"syscall"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/rio"
)

var _ rio.Placer = BindPlacer

/*
	Gets material from srcPath to destPath by use of a bind mount.

	Requesting a read-only result will be honored.

	Direct mounts are cannot be supported by this placer, and requesting one will error.

	May panic with:

	  - `*rio.ErrAssembly` -- for any show-stopping IO errors.
	  - `*rio.ErrAssembly` -- if given paths that are not plain dirs.
*/
func BindPlacer(srcPath, destPath string, writable bool, _ bool) rio.Emplacement {
	sys := "bindplacer" // label in logs and errors.
	mustBeDir(sys, srcPath, "srcPath")
	mustBeDir(sys, destPath, "destPath")
	flags := syscall.MS_BIND | syscall.MS_REC
	if err := syscall.Mount(srcPath, destPath, "bind", uintptr(flags), ""); err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sys},
			meep.Cause(err),
		))
	}
	if !writable {
		flags |= syscall.MS_RDONLY | syscall.MS_REMOUNT
		if err := syscall.Mount(srcPath, destPath, "bind", uintptr(flags), ""); err != nil {
			panic(meep.Meep(
				&rio.ErrAssembly{System: sys},
				meep.Cause(err),
			))
		}
	}
	return bindEmplacement{path: destPath}
}

type bindEmplacement struct {
	path string
}

func (e bindEmplacement) Teardown() {
	if err := syscall.Unmount(e.path, 0); err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: "bindplacer", Path: "teardown"},
			meep.Cause(err),
		))
	}
}

func mustBeDir(sysLabel string, pth string, callIt string) {
	stat, err := os.Stat(pth)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sysLabel, Path: callIt},
			meep.Cause(err),
		))
	}
	if !stat.IsDir() {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sysLabel, Path: callIt},
			meep.Cause(fmt.Errorf("must be dir")),
		))
	}
}
