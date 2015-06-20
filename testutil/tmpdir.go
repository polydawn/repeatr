package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
)

/*
	Decorates a goconvey test with a tmpdir.

	See also https://github.com/smartystreets/goconvey/wiki/Decorating-tests-to-provide-common-logic
*/
func WithTmpdir(fn func()) func() {
	return func() {
		retreat, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		if originalDir == "" {
			originalDir = retreat
		}

		convey.Reset(func() {
			os.Chdir(retreat)
		})

		tmpBase := def.Base()
		err = os.MkdirAll(tmpBase, 0755)
		if err != nil {
			panic(err)
		}
		err = os.Chdir(tmpBase)
		if err != nil {
			panic(err)
		}

		err = os.MkdirAll("repeatr-test", 1755)
		if err != nil {
			panic(err)
		}
		tmpdir, err := ioutil.TempDir("repeatr-test", "")
		if err != nil {
			panic(err)
		}
		tmpdir, err = filepath.Abs(tmpdir)
		if err != nil {
			panic(err)
		}
		convey.Reset(func() {
			os.RemoveAll(tmpdir)
		})
		err = os.Chdir(tmpdir)
		if err != nil {
			panic(err)
		}

		fn()
	}
}

/*
	Returns the first cwd, before anyone ever `WithTmpdir`'d.

	For most tests, this will be the test package directory in the source tree.
*/
func OriginalDir() string {
	// not goroutine safe, but what kind of maniac are you anyway?
	if originalDir == "" {
		var err error
		originalDir, err = os.Getwd()
		if err != nil {
			panic(err)
		}
	}
	return originalDir
}

var originalDir string
