package config

import (
	"os"
	"path/filepath"

	"go.polydawn.net/rio/fs"
)

/*
	Return the path to a dir that will be used to read memoization of
	previous runs -- enabling short-circuit returns if they're encountered
	again -- and also used as the place to record a memo of this run.

	The default value is nil -- there will be no memoization --
	and this can be set by the `REPEATR_MEMODIR` environment variable.
*/
func GetRepeatrMemoPath() *fs.AbsolutePath {
	pth := os.Getenv("REPEATR_MEMODIR")
	if pth == "" {
		return nil
	}
	pth, err := filepath.Abs(pth)
	if err != nil {
		panic(err)
	}
	memoDir := fs.MustAbsolutePath(pth)
	return &memoDir
}
