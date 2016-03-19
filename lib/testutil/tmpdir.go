package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/smartystreets/goconvey/convey"
)

/*
	Decorates a goconvey test with a tmpdir.

	See also https://github.com/smartystreets/goconvey/wiki/Decorating-tests-to-provide-common-logic
*/
func WithTmpdir(fn interface{}) func(c convey.C) {
	return func(c convey.C) {
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

		tmpBase := "/tmp/repeatr-test/"
		err = os.MkdirAll(tmpBase, os.FileMode(0755)|os.ModeSticky)
		if err != nil {
			panic(err)
		}
		err = os.Chdir(tmpBase)
		if err != nil {
			panic(err)
		}

		tmpdir, err := ioutil.TempDir("", "")
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

		switch fn := fn.(type) {
		case func():
			fn()
		case func(c convey.C):
			fn(c)
		}
	}
}

/*
	Calls `fn` after chdir'ing to `dir`, and resets chdir on return.

	`dir` is expected to already exist, and will not be removed on return.
*/
func UsingDir(dir string, fn func()) {
	retreat, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if originalDir == "" {
		originalDir = retreat
	}

	defer os.Chdir(retreat)

	err = os.Chdir(dir)
	if err != nil {
		panic(err)
	}

	fn()
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
