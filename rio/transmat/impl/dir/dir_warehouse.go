package dir

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"syscall"

	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/rio"
)

/*
	Refer to, interact with, and manage a warehouse.

	Warehouses are generally figured to be tolerant of group management,
	and this one is no different: operations that are atomic in the face
	of multiple uncoordinated actions are the default.
	There are no daemons involved.
*/
type Warehouse struct {
	localPath string
	ctntAddr  bool
}

/*
	Initialize a warehouse controller.

	May panic with:
	  - Config Error: if the URI is unparsable or has an unsupported scheme.
*/
func NewWarehouse(coords rio.SiloURI) *Warehouse {
	wh := &Warehouse{}
	u, err := url.Parse(string(coords))
	if err != nil {
		panic(rio.ConfigError.New("failed to parse URI: %s", err))
	}
	switch u.Scheme {
	case "file+ca":
		wh.ctntAddr = true
		fallthrough
	case "file":
		wh.localPath = filepath.Join(u.Host, u.Path) // file uris don't have hosts
	case "":
		panic(rio.ConfigError.New("missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\""))
	default:
		panic(rio.ConfigError.New("unsupported scheme in warehouse URI: %q", u.Scheme))
	}
	return wh
}

/*
	Check if the warehouse exists and can be contacted.
*/
func (wh *Warehouse) Ping() bool {
	expectedDirPath := wh.localPath
	if !wh.ctntAddr {
		expectedDirPath = filepath.Dir(expectedDirPath)
	}
	stat, err := os.Stat(expectedDirPath)
	// don't particularly care if `os.IsNotExist`; any error means it's probably not usable.
	return err == nil && stat.IsDir()
}

/*
	Return the (local) path expected for a given piece of data.
*/
func (wh *Warehouse) GetShelf(dataHash rio.CommitID) string {
	if wh.ctntAddr {
		return filepath.Join(wh.localPath, string(dataHash))
	} else {
		return wh.localPath
	}
}

type writeController struct {
	warehouse *Warehouse
	tmpPath   string
}

func (wh *Warehouse) openWriter() *writeController {
	wc := &writeController{}
	wc.warehouse = wh
	wc.tmpPath = wc.claimPrecommitPath()
	return wc
}

/*
	Returns a local file path to a tempdir for writing pre-commit data to.

	May panic with:
	  - WarehouseConnectionError: if we can't write; since tempdir is
	      created using the appropriate atomic mechanisms, we do touch
	      the disk before returning the path string.
*/
func (wc *writeController) claimPrecommitPath() string {
	var precommitPath string
	var err error
	if wc.warehouse.ctntAddr {
		precommitPath, err = ioutil.TempDir(
			wc.warehouse.localPath,
			".tmp.upload."+guid.New()+".",
		)
	} else {
		precommitPath, err = ioutil.TempDir(
			filepath.Dir(wc.warehouse.localPath),
			".tmp.upload."+filepath.Base(wc.warehouse.localPath)+"."+guid.New()+".",
		)
	}
	if err != nil {
		panic(rio.WarehouseIOError.New("failed to reserve temp space in warehouse: %s", err))
	}
	return precommitPath
}

/*
	Commit the current data as the given hash.
	Caller must be an adult and specify the hash truthfully.
	Closes the writer and invalidates any future use.
*/
func (wc *writeController) commit(saveAs rio.CommitID) {
	// This is a rather alarming flow chart and almost makes me want to
	//  split apart the implementations for CA and non-CA entirely,
	//   but here goes:
	// if CA:
	//  - attempt to move
	//  - if exists, okay.  rm yourself.
	// if non-CA:
	//  - if exists, push it aside.
	//  - attempt to move.
	//  - if exists, store error
	//  - remove either the old pushedaside, or yourself (... or not, if wanted for debug).

	destPath := wc.warehouse.GetShelf(saveAs)
	if wc.warehouse.ctntAddr {
		err := os.Rename(wc.tmpPath, destPath)
		// if success, it's committed; yayy, we're done
		if err == nil {
			return
		}
		// if there was already a dir there, another actor committed the same thing in a race,
		//  which is fine: we'll just see ourselves out.
		if err2 := err.(*os.LinkError); err2.Err == syscall.ENOTEMPTY {
			os.RemoveAll(wc.tmpPath)
			return
		}
		// any other errors are quite alarming
		panic(rio.WarehouseIOError.New("failed to commit %s: %s", saveAs, err))
	} else {
		pushedAside := pushAside(destPath)
		err := os.Rename(wc.tmpPath, destPath)
		if err != nil {
			// In non-CA mode, this should only happen in case of misconfig or problems from
			// racey use (in which case as usual, you're already Doing It Wrong and we're just being frank about it).
			panic(rio.WarehouseIOError.New("failed moving data to committed location: %s", err))
		}
		// Clean up.
		//  When not using a CA mode, this involves destroying the
		//   previous data -- we don't want crap building up indefinitely,
		//   right? -- and while this is no different than, say, tar's
		//   behavior, it's also a little scary.  Use CA mode, ffs.
		os.RemoveAll(pushedAside)
	}

}

// kick the thing to a sibling path of itself.
func pushAside(obstructionPath string) string {
	base := filepath.Base(obstructionPath)
	dir := filepath.Dir(obstructionPath)
	var err error
	for i := 0; i < 100; i++ {
		try := filepath.Join(
			dir,
			".tmp.expired."+base+"."+guid.New(),
		)
		err = os.Rename(obstructionPath, try)
		if err == nil {
			return try
		} else if os.IsNotExist(err) {
			return ""
		}
		// I can't see a reasonable way to check if the move failed
		// because the destination path already existed (thank you,
		// golang stdlib for throwing us all the fucking way to
		// platform specific syscall magic values because you couldn't
		// be arsed to formulate a useful concept of errors)... so,
		// fuck it, we won't.  We'll just spin on our hands for a while.
		// Things like permission errors will be reported... after 99
		// doomed-to-fail tries more than necessary, because stdlib
		// can't be arsed to normalize errors and I'm too mad to do it.
		// This is the same thing ioutil.TempDir does.  And that makes me sad.
	}
	panic(rio.WarehouseIOError.New("failed evicting old data from commit location: %s", err))
}
