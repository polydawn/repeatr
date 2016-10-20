package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/rio"
)

func mustDir(pth string) {
	if err := os.MkdirAll(pth, 0755); err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to set up workspace"},
			meep.Cause(err),
		))
	}
}

type workArea struct {
	fullCheckouts  string
	nosubCheckouts string
	gitDirs        string
}

func (wa workArea) gitDirPath(repoURL string) string {
	return filepath.Join(wa.gitDirs, slugifyRemote(repoURL))
}

func (wa workArea) makeFullchTempPath(commitHash string) string {
	pth, err := ioutil.TempDir(wa.fullCheckouts, commitHash+"-")
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to set up tempdir"},
			meep.Cause(err),
		))
	}
	return pth
}

func (wa workArea) getFullchFinalPath(commitHash string) string {
	return filepath.Join(wa.fullCheckouts, commitHash)
}

func (wa workArea) makeNosubchTempPath(commitHash string) string {
	pth, err := ioutil.TempDir(wa.nosubCheckouts, commitHash+"-")
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to set up tempdir"},
			meep.Cause(err),
		))
	}
	return pth
}

func (wa workArea) getNosubchFinalPath(commitHash string) string {
	return filepath.Join(wa.nosubCheckouts, commitHash)
}

func moveOrShrug(from string, to string) {
	err := os.Rename(from, to)
	if err != nil {
		if err2, ok := err.(*os.LinkError); ok &&
			err2.Err == syscall.EBUSY || err2.Err == syscall.ENOTEMPTY {
			// oh, fine.  somebody raced us to it.  we believe in them: just return.
			return
		}
		panic(meep.Meep(
			&rio.ErrInternal{Msg: fmt.Sprintf("Error commiting %q into cache", from)},
			meep.Cause(err),
		))
	}
}
