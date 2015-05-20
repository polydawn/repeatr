package flak

import (
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
)

// Runs a function with a tempdir, cleaning up afterward.
func WithDir(f func(string), dirs ...string) {

	if len(dirs) < 1 {
		panic(errors.ProgrammerError.New("Must have at least one sub-folder for tempdir"))
	}

	tempPath := filepath.Join(dirs...)

	// Tempdir wants parent path to exist
	err := os.MkdirAll(tempPath, 0755)
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
