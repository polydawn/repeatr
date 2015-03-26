package flak

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
)

// Methods that many executors might use

// Generates a temporary repeatr directory, creating all neccesary parent folders.
// Must be passed at least one directory name, all of which will be used in the path.
// Uses os.TempDir() to decide where to place.
//
// For example, GetTempDir("my-executor") -> /tmp/repeatr/my-executor/989443394
func GetTempDir(dirs ...string) string {

	if len(dirs) < 1 {
		panic(errors.ProgrammerError.New("Must have at least one sub-folder for tempdir"))
	}

	dir := []string{os.TempDir(), "repeatr"}
	dir = append(dir, dirs...)
	tempPath := filepath.Join(dir...)

	// Tempdir wants parent path to exist
	err := os.MkdirAll(tempPath, 0600)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	// Make temp dir for this instance
	folder, err := ioutil.TempDir(tempPath, "")
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	return folder
}

// Runs a function with a tempdir, cleaning up afterward.
func WithTempDir(f func(string), dirs ...string) {

	if len(dirs) < 1 {
		panic(errors.ProgrammerError.New("Must have at least one sub-folder for tempdir"))
	}

	tempPath := filepath.Join(dirs...)

	// Tempdir wants parent path to exist
	err := os.MkdirAll(tempPath, 0600)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	try.Do(func() {
		f(tempPath)
	}).Finally(func() {
		err := os.RemoveAll(tempPath)
		if err != nil {
			// TODO: we don't want to panic here, more like a debug log entry, "failed to remove tempdir."
			// Can accomplish once we add logging.
			panic(errors.IOError.Wrap(err))
		}
	}).Done()
}
