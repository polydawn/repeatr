package flak

import (
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
)

/*
	Creates a directory, calls the given function, and removes the dir again afterward.
	The created path is handed to the given function (cwd is untouched).

	Multiple parent dirs will be created if necessary.  These will not be
	removed afterwards.

	If the function panics, the dir will *not* be removed.  (We're using this
	in the executor code (n.b. todo refactor it to that package) where mounts
	are flying: we strongly want to avoid a recursive remove in case an error
	was raised from mount cleanup!)
*/
func WithDir(f func(string), dirs ...string) {
	// Mkdirs
	if len(dirs) < 1 {
		panic(errors.ProgrammerError.New("WithDir must have at least one sub-directory"))
	}
	tempPath := filepath.Join(dirs...)
	err := os.MkdirAll(tempPath, 0755)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	// Lambda
	f(tempPath)

	// Cleanup
	//  this is intentionally not in a defer or try/finally -- it's critical we *don't* do this for all errors.
	//  specifically, if there's a placer error?  hooooly shit DO NOT proceed on a bunch of deletes;
	//  in a worst case scenario that placer error might have been failure to remove a bind from the host.
	//  and that would leave a wormhole straight to hell which we should really NOT pump energy into.
	err = os.RemoveAll(tempPath)
	if err != nil {
		// TODO: we don't want to panic here, more like a debug log entry, "failed to remove tempdir."
		// Can accomplish once we add logging.
		panic(errors.IOError.Wrap(err))
	}
}
