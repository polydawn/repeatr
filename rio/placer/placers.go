package placer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/lib/fspatch"
	"go.polydawn.net/repeatr/rio"
)

var _ rio.Placer = CopyingPlacer

func CopyingPlacer(srcBasePath, destBasePath string, _ bool, bareMount bool) rio.Emplacement {
	srcBaseStat, err := os.Stat(srcBasePath)
	if err != nil || !srcBaseStat.IsDir() {
		panic(Error.New("copyingplacer: srcPath %q must be dir: %s", srcBasePath, err))
	}
	destBaseStat, err := os.Stat(destBasePath)
	if err != nil || !destBaseStat.IsDir() {
		panic(Error.New("copyingplacer: destPath %q must be dir: %s", destBasePath, err))
	}
	if bareMount {
		panic(Error.New("copyingplacer: can't support doing a bare mount with this placer; you'll to pick a more powerful one"))
	}
	// remove any files already here (to emulate behavior like an overlapping mount)
	// (can't take the easy route and just `os.RemoveAll(destBasePath)` because that propagates times changes onto the parent.)
	d, err := os.Open(destBasePath)
	if err != nil {
		panic(Error.New("copyingplacer: io error: %s", err))
	}
	names, err := d.Readdirnames(-1)
	if err != nil {
		panic(Error.New("copyingplacer: io error: %s", err))
	}
	for _, name := range names {
		err := os.RemoveAll(filepath.Join(destBasePath, name))
		if err != nil {
			panic(Error.New("copyingplacer: io error: %s", err))
		}
	}
	// walk and copy
	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}
		hdr, file := fs.ScanFile(srcBasePath, filenode.Path, filenode.Info)
		if file != nil {
			defer file.Close()
		}
		fs.PlaceFile(destBasePath, hdr, file)
		return nil
	}
	postVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Info.IsDir() {
			if err := fspatch.UtimesNano(filepath.Join(destBasePath, filenode.Path), fs.Epochwhen, filenode.Info.ModTime()); err != nil {
				return err
			}
		}
		return nil
	}
	try.Do(func() {
		if err := fs.Walk(srcBasePath, preVisit, postVisit); err != nil {
			panic(err)
		}
	}).CatchAll(func(err error) {
		panic(Error.New("copyingplacer: io failed: %s", err))
	}).Done()

	return copyEmplacement{path: destBasePath}
}

type copyEmplacement struct {
	path string
}

func (e copyEmplacement) Teardown() {
	if err := os.RemoveAll(e.path); err != nil {
		panic(Error.New("copyingplacer: teardown failed: %s", err))
	}
}

var _ rio.Placer = BindPlacer

func BindPlacer(srcPath, destPath string, writable bool, _ bool) rio.Emplacement {
	srcStat, err := os.Stat(srcPath)
	if err != nil || !srcStat.IsDir() {
		panic(Error.New("bindplacer: srcPath %q must be dir: %s", srcPath, err))
	}
	destStat, err := os.Stat(destPath)
	if err != nil || !destStat.IsDir() {
		panic(Error.New("bindplacer: destPath %q must be dir: %s", destPath, err))
	}
	flags := syscall.MS_BIND | syscall.MS_REC
	if err := syscall.Mount(srcPath, destPath, "bind", uintptr(flags), ""); err != nil {
		panic(Error.New("bindplacer: bind error: %s", err))
	}
	if !writable {
		flags |= syscall.MS_RDONLY | syscall.MS_REMOUNT
		if err := syscall.Mount(srcPath, destPath, "bind", uintptr(flags), ""); err != nil {
			panic(Error.New("bindplacer: bind error: %s", err))
		}
	}
	return bindEmplacement{path: destPath}
}

type bindEmplacement struct {
	path string
}

func (e bindEmplacement) Teardown() {
	if err := syscall.Unmount(e.path, 0); err != nil {
		panic(Error.New("bindplacer: teardown failed: %s", err))
	}
}

func NewAufsPlacer(workPath string) rio.Placer {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}
	workPath, err = filepath.Abs(workPath)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}
	return func(srcBasePath, destBasePath string, writable bool, bareMount bool) rio.Emplacement {
		srcBaseStat, err := os.Stat(srcBasePath)
		if err != nil || !srcBaseStat.IsDir() {
			panic(Error.New("aufsplacer: srcPath %q must be dir: %s", srcBasePath, err))
		}
		destBaseStat, err := os.Stat(destBasePath)
		if err != nil || !destBaseStat.IsDir() {
			panic(Error.New("aufsplacer: destPath %q must be dir: %s", destBasePath, err))
		}
		// if a RO mount is requested, no need to set up COW; just hand off to bind.
		if !writable {
			return BindPlacer(srcBasePath, destBasePath, writable, bareMount)
		}
		// similarly, if the caller intentionally wants a bare mount, no need for COW; just hand off.
		if bareMount {
			return BindPlacer(srcBasePath, destBasePath, writable, bareMount)
		}
		// make work dir for the overlay layer
		layerPath, err := ioutil.TempDir(workPath, "layer-")
		if err != nil {
			panic(errors.IOError.Wrap(err))
		}
		// set up COW
		// if you were doing this in a shell, it'd be roughly `mount -t aufs -o br="$layer":"$base" none "$composite"`.
		// yes, this may behave oddly in the event of paths containing ":" or "=".
		syscall.Mount("none", destBasePath, "aufs", 0, fmt.Sprintf("br:%s=rw:%s=ro", layerPath, srcBasePath))
		// fix props on layerPath; otherwise they instantly leak through
		hdr, _ := fs.ScanFile(srcBasePath, "./", srcBaseStat)
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
		panic(Error.New("aufsplacer: teardown failed: %s", err))
	}
	// now throw away the layer contents
	if err := os.RemoveAll(e.layerPath); err != nil {
		panic(Error.New("aufsplacer: teardown failed: %s", err))
	}
}
