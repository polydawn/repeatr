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
	"go.polydawn.net/repeatr/rio/placer/impl/copy"
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
		// if it's a file mount, aufs doesn't apply, so we have to shell out to copy.
		wantMode := getSrcMode(srcBasePath, sys)
		if wantMode == 0 {
			return copy.CopyingPlacer(srcBasePath, destBasePath, writable, bareMount)
		}
		mkDest(destBasePath, wantMode, sys)
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

func getSrcMode(srcBasePath string, logLabel string) os.FileMode {
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
}

func mkDest(destBasePath string, wantMode os.FileMode, logLabel string) {
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
