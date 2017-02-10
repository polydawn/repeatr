// +build linux

package bind

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/rio"
)

var _ rio.Placer = BindPlacer

/*
	Gets material from srcPath to destPath by use of a bind mount.

	Path parameters should be absolute.
	(If we need to create files or dirs for the dest, we will also
	then re-fix the modified times from above.)

	Requesting a read-only result will be honored.

	Direct mounts are always what this placer provides,
	so the parameter is ignored.

	May panic with:

	  - `*rio.ErrAssembly` -- for any show-stopping IO errors.
	  - `*rio.ErrAssembly` -- if given paths that are not plain dirs.
*/
func BindPlacer(srcBasePath, destPath string, writable bool, _ bool) rio.Emplacement {
	sys := "bindplacer" // label in logs and errors.

	// Make the destination path exist and be the right type to mount over.
	mkDest(srcBasePath, destPath, sys)

	// Make mount syscall to bind, and optionally then push it to readonly.
	// Works the same for dirs or files.
	flags := syscall.MS_BIND | syscall.MS_REC
	if err := syscall.Mount(srcBasePath, destPath, "bind", uintptr(flags), ""); err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sys},
			meep.Cause(err),
		))
	}
	if !writable {
		flags |= syscall.MS_RDONLY | syscall.MS_REMOUNT
		if err := syscall.Mount(srcBasePath, destPath, "bind", uintptr(flags), ""); err != nil {
			panic(meep.Meep(
				&rio.ErrAssembly{System: sys},
				meep.Cause(err),
			))
		}
	}
	return bindEmplacement{path: destPath}
}

func mkDest(srcBasePath, destBasePath string, logLabel string) {
	// Determine desired type.
	wantMode := func() os.FileMode {
		srcBaseStat, err := os.Stat(srcBasePath)
		if err != nil {
			panic(meep.Meep(
				&rio.ErrAssembly{System: logLabel, Path: "srcPath"},
				meep.Cause(err),
			))
		}
		mode := srcBaseStat.Mode() & os.ModeType
		switch mode {
		case os.ModeDir, 0:
			return mode
		default:
			panic(meep.Meep(
				&rio.ErrAssembly{System: logLabel, Path: "srcPath"},
				meep.Cause(fmt.Errorf("source may only be dir or plain file")),
			))
		}
	}()

	// Handle all the cases for existing things at destination.
	destBaseStat, err := os.Stat(destBasePath)
	if err == nil {
		// If exists and wrong type, ErrAssembly.
		if destBaseStat.Mode()&os.ModeType != wantMode {
			panic(meep.Meep(
				&rio.ErrAssembly{System: logLabel, Path: "destPath"},
				meep.Cause(fmt.Errorf("already exists and is different type than source")),
			))
		}
		// If exists and right type, exit early.
		return
	}
	// If it doesn't exist, that's fine; any other error, ErrAssembly.
	if !os.IsNotExist(err) {
		panic(meep.Meep(
			&rio.ErrAssembly{System: logLabel, Path: "destPath"},
			meep.Cause(err),
		))
	}

	// If we made it this far: dest doesn't exist yet.
	// Capture the parent dir mtime, because we're about to disrupt it.

	// Make the dest node, matching type of the source.
	// The perms don't matter; will be shadowed.
	// We assume the parent dirs are all in place because you're almost
	// certainly using this as part of an assembler.
	fs.WithMtimeRepair(filepath.Dir(destBasePath), func() {
		switch wantMode {
		case os.ModeDir:
			err = os.Mkdir(destBasePath, 0644)
		case 0:
			var f *os.File
			f, err = os.OpenFile(destBasePath, os.O_CREATE, 0644)
			f.Close()
		}
		if err != nil {
			panic(meep.Meep(
				&rio.ErrAssembly{System: logLabel, Path: "destPath"},
				meep.Cause(err),
			))
		}
	})
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
