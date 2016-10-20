package placer

import (
	"fmt"
	"io/ioutil"
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

func NewAufsPlacer(workPath string) rio.Placer {
	sys := "aufsplacer" // label in logs and errors.
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sys, Path: "workdir"},
			meep.Cause(err),
		))
	}
	workPath, err = filepath.Abs(workPath)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sys, Path: "workdir"},
			meep.Cause(err),
		))
	}
	return func(srcBasePath, destBasePath string, writable bool, bareMount bool) rio.Emplacement {
		// if a RO mount is requested, no need to set up COW; just hand off to bind.
		if !writable {
			return BindPlacer(srcBasePath, destBasePath, writable, bareMount)
		}
		// similarly, if the caller intentionally wants a bare mount, no need for COW; just hand off.
		if bareMount {
			return BindPlacer(srcBasePath, destBasePath, writable, bareMount)
		}
		// ok, we really have to work.  validate params.
		mustBeDir(sys, srcBasePath, "srcPath")
		mustBeDir(sys, destBasePath, "destPath")
		// make work dir for the overlay layer
		layerPath, err := ioutil.TempDir(workPath, "layer-")
		if err != nil {
			panic(meep.Meep(
				&rio.ErrAssembly{System: sys, Path: "layerDir"},
				meep.Cause(err),
			))
		}
		// set up COW
		// if you were doing this in a shell, it'd be roughly `mount -t aufs -o br="$layer":"$base" none "$composite"`.
		// yes, this may behave oddly in the event of paths containing ":" or "=".
		if err := syscall.Mount("none", destBasePath, "aufs", 0, fmt.Sprintf("br:%s=rw:%s=ro", layerPath, srcBasePath)); err != nil {
			panic(meep.Meep(
				&rio.ErrAssembly{System: sys},
				meep.Cause(err),
			))
		}
		// fix props on layerPath; otherwise they instantly leak through
		hdr, _ := fs.ScanFile(srcBasePath, "./")
		fs.PlaceFile(layerPath, hdr, nil)
		// that's it; setting up COW also mounted it into destination.
		return aufsEmplacement{
			layerPath:   layerPath,
			landingPath: destBasePath,
		}
	}
}

type aufsEmplacement struct {
	layerPath   string
	landingPath string
}

func (e aufsEmplacement) Teardown() {
	// first tear down the mount
	if err := syscall.Unmount(e.landingPath, 0); err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: "aufsplacer", Path: "teardown"},
			meep.Cause(err),
		))
	}
	// now throw away the layer contents
	if err := os.RemoveAll(e.layerPath); err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: "aufsplacer", Path: "teardown"},
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
