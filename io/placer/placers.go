package placer

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fspatch"
)

var _ integrity.Placer = CopyingPlacer

func CopyingPlacer(srcBasePath, destBasePath string, _ bool) integrity.Emplacement {
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
			if err := fspatch.UtimesNano(filepath.Join(destBasePath, filenode.Path), def.Epochwhen, filenode.Info.ModTime()); err != nil {
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
		panic(Error.New("copyingplacer: io failed: ", err))
	}).Done()

	return copyEmplacement{path: destBasePath}
}

type copyEmplacement struct {
	path string
}

func (e copyEmplacement) Teardown() {
	if err := os.RemoveAll(e.path); err != nil {
		panic(Error.New("copyingplacer: teardown failed: ", err))
	}
}

var _ integrity.Placer = BindPlacer

func BindPlacer(srcPath, destPath string, writable bool) integrity.Emplacement {
	srcStat, err := os.Stat(srcPath)
	if err != nil || !srcStat.IsDir() {
		panic(Error.New("bindplacer: srcPath %q must be dir: %s", srcPath, err))
	}
	destStat, err := os.Stat(destPath)
	if err != nil || !destStat.IsDir() {
		panic(Error.New("bindplacer: destPath %q must be dir: %s", destPath, err))
	}
	flags := syscall.MS_BIND | syscall.MS_REC
	if !writable {
		flags |= syscall.MS_RDONLY
	}
	if err := syscall.Mount(srcPath, destPath, "bind", uintptr(flags), ""); err != nil {
		panic(Error.New("bindplacer: bind error: %s", err))
	}
	return bindEmplacement{path: destPath}
}

type bindEmplacement struct {
	path string
}

func (e bindEmplacement) Teardown() {
	if err := syscall.Unmount(e.path, 0); err != nil {
		panic(Error.New("bindplacer: teardown failed: ", err))
	}
}

var _ integrity.Placer = AufsPlacer

func AufsPlacer(srcPath, destPath string, writable bool) integrity.Emplacement {
	return nil
}