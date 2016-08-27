// +build linux
package placer

import (
        "os"
        "path/filepath"

        "github.com/spacemonkeygo/errors/try"
        "go.polydawn.net/repeatr/lib/fs"
        "go.polydawn.net/repeatr/lib/fspatch"
        "go.polydawn.net/repeatr/rio"
)

var _ rio.Placer = CopyingPlacer

func CopyingPlacer(srcBasePath, destBasePath string, _ bool, bareMount bool) rio.Emplacement {
        srcBaseStat, err := os.Stat(srcBasePath)
        if err != nil {
                panic(Error.New("copyingplacer: could not stat srcPath %s: %s", srcBasePath, err))
        }
        _, err = os.Stat(destBasePath)
        if err != nil && !os.IsNotExist(err) {
                panic(Error.New("copyingplacer: could not stat destPath %s: %s", destBasePath, err))
        }
        if bareMount {
                panic(Error.New("copyingplacer: can't support doing a bare mount with this placer; you'll to pick a more powerful one"))
        }
        // remove any files already here (to emulate behavior like an overlapping mount)
        // also, reject any destinations of the wrong type
        typ := srcBaseStat.Mode() & os.ModeType
        switch typ {
        case os.ModeDir:
                if !os.IsNotExist(err) {
                        // can't take the easy route and just `os.RemoveAll(destBasePath)`
                        //  because that propagates times changes onto the parent.
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
                }
        case 0:
                // Files: easier.
                hdr, body := fs.ScanFile(srcBasePath, "", srcBaseStat)
                defer body.Close()
                fs.PlaceFile(destBasePath, hdr, body)
        default:
                panic(Error.New("copyingplacer: destPath may only be dirs or files"))
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
	panic("BindPlacer unsupported on darwin")
}

type bindEmplacement struct {
        path string
}

func (e bindEmplacement) Teardown() {

}

func NewAufsPlacer(workPath string) rio.Placer {
	panic("AufsPlacer unsupported on darwin")
}

type aufsEmplacement struct {
        layerPath   string
        landingPath string
}

func (e aufsEmplacement) Teardown() {
}
