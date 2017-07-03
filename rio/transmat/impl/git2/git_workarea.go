package git2

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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
	fullCheckouts        string
	noSubmoduleCheckouts string
	gitStorageDirs       string
}

func (wa workArea) gitStorageDirPath(repoURL string) string {
	return filepath.Join(wa.gitStorageDirs, slugifyRemote(repoURL))
}

func (wa workArea) makeFullCheckoutTempPath(commitHash string) string {
	pth, err := ioutil.TempDir(wa.fullCheckouts, commitHash+"-")
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to set up tempdir"},
			meep.Cause(err),
		))
	}
	return pth
}

func (wa workArea) getFullCheckoutFinalPath(commitHash string) string {
	return filepath.Join(wa.fullCheckouts, commitHash)
}

func (wa workArea) makeNoSubmoduleCheckoutTempPath(commitHash string) string {
	pth, err := ioutil.TempDir(wa.noSubmoduleCheckouts, commitHash+"-")
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to set up tempdir"},
			meep.Cause(err),
		))
	}
	return pth
}

func (wa workArea) getNosubchFinalPath(commitHash string) string {
	return filepath.Join(wa.noSubmoduleCheckouts, commitHash)
}

func moveOrShrug(from string, to string) {
	err := os.Rename(from, to)
	if err != nil {
		if _, ok := err.(*os.LinkError); ok && os.IsExist(err) {
			// oh, fine.  somebody raced us to it.  we believe in them: just return.
			return
		}
		panic(meep.Meep(
			&rio.ErrInternal{Msg: fmt.Sprintf("Error commiting %q into cache", from)},
			meep.Cause(err),
		))
	}
}
