package flak

import (
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Methods that many executors might use

// Generates a temporary repeatr directory, creating all neccesary parent folders.
// Must be passed at least one directory name, all of which will be used in the path.
// Uses os.TempDir() to decide where to place.
//
// For example, GetTempDir("my-executor") -> /tmp/repeatr/my-executor/989443394
func GetTempDir(dirs ...string) string {

	if len(dirs) < 1 {
		Println("Must have at least one sub-folder for tempdir", "replace with space monkey")
		panic("whelp")
	}

	dir := []string{os.TempDir(), "repeatr"}
	dir = append(dir, dirs...)
	tempPath := filepath.Join(dir...)

	// Tempdir wants parent path to exist
	err := os.MkdirAll(tempPath, 0600)
	if err != nil {
		Println(err, "replace with space monkey")
	}

	// Make temp dir for this instance
	folder, err := ioutil.TempDir(tempPath, "")
	if err != nil {
		Println(err, "replace with space monkey")
	}

	return folder
}
