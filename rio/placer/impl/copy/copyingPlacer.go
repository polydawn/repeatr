package copy

import (
	"fmt"
	"os"
	"path/filepath"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/lib/fspatch"
	"go.polydawn.net/repeatr/rio"
)

var _ rio.Placer = CopyingPlacer

/*
	Gets material from srcPath to destPath by implementing a recursive copy.

	Whether you need a "read-only" (fork) or not is ignored; you're getting one.
	The result filesystem will always be writable; it is not possible to make
	a read-only filesystem with this placer.

	Direct mounts cannot be supported by this placer, and requesting one will error.

	May panic with:

	  - `*rio.ErrAssembly` -- for any show-stopping IO errors.
	  - `*rio.ErrAssembly` -- if given paths that are not plain files or dirs.
	  - `*def.ErrConfigValidation` -- if requesting a direct mount, which is unsupported.
*/
func CopyingPlacer(srcBasePath, destBasePath string, _ bool, bareMount bool) rio.Emplacement {
	sys := "copyingplacer" // label in logs and errors.
	srcBaseStat, err := os.Stat(srcBasePath)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sys, Path: "srcPath"},
			meep.Cause(err),
		))
	}
	_, err = os.Stat(destBasePath)
	if err != nil && !os.IsNotExist(err) {
		panic(meep.Meep(
			&rio.ErrAssembly{System: sys, Path: "destPath"},
			meep.Cause(err),
		))
	}
	if bareMount {
		panic(
			&def.ErrConfigValidation{Msg: sys + " can't support doing a direct mount; you'll to pick a more powerful one"},
		)
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
				panic(meep.Meep(
					&rio.ErrAssembly{System: sys, Path: "destPath"},
					meep.Cause(err),
				))
			}
			names, err := d.Readdirnames(-1)
			if err != nil {
				panic(meep.Meep(
					&rio.ErrAssembly{System: sys, Path: "destPath"},
					meep.Cause(err),
				))
			}
			for _, name := range names {
				err := os.RemoveAll(filepath.Join(destBasePath, name))
				if err != nil {
					panic(meep.Meep(
						&rio.ErrAssembly{System: sys, Path: "destPath"},
						meep.Cause(err),
					))
				}
			}
		}
	case 0:
		// Files: easier.
		hdr, body := fs.ScanFile(srcBasePath, "", srcBaseStat)
		defer body.Close()
		fs.PlaceFile(destBasePath, hdr, body)
	default:
		panic(meep.Meep(
			&rio.ErrAssembly{System: sys, Path: "destPath"},
			meep.Cause(fmt.Errorf("destination may only be dir or plain file")),
		))
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
	err = fs.Walk(srcBasePath, preVisit, postVisit)
	meep.TryPlan{
		{CatchAny: true, Handler: func(e error) {
			panic(meep.Meep(
				&rio.ErrAssembly{System: sys},
				meep.Cause(err),
			))
		}},
	}.MustHandle(err)

	return copyEmplacement{path: destBasePath}
}

type copyEmplacement struct {
	path string
}

func (e copyEmplacement) Teardown() {
	if err := os.RemoveAll(e.path); err != nil {
		panic(meep.Meep(
			&rio.ErrAssembly{System: "copyingplacer", Path: "teardown"},
			meep.Cause(err),
		))
	}
}
