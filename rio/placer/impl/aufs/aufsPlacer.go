// +build linux

package aufs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/rio"
	"go.polydawn.net/repeatr/rio/placer/impl/bind"
)

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
			return bind.BindPlacer(srcBasePath, destBasePath, writable, bareMount)
		}
		// similarly, if the caller intentionally wants a bare mount, no need for COW; just hand off.
		if bareMount {
			return bind.BindPlacer(srcBasePath, destBasePath, writable, bareMount)
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
