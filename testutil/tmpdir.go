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
func WithTmpdir(fn func()) func() {
	return func() {
		retreat, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		convey.Reset(func() {
			os.Chdir(retreat)
		})

		tmpBase := os.Getenv("TMPDIR")
		if len(tmpBase) == 0 {
			tmpBase = os.TempDir()
		}
		err = os.Chdir(tmpBase)
		if err != nil {
			panic(err)
		}

		err = os.MkdirAll("repeatr-test", 0755)
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
