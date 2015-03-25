/*
	Fixtures for use in testing.

	Do not import this in non-test code.

	Upon import, this creates many files -- and does not clean them up.
	Your test package must call `fixtures.Cleanup` after all tests, like this:

		func TestMain(m *testing.M) {
			code := m.Run()
			inputfixtures.Cleanup()
			os.Exit(code)
		}

	Note that use of `dir` inputs, as usual, requires root on the host so that
	file ownership can be forced to standardized values.
*/
package inputfixtures

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/flak"
)

var basePath string = flak.GetTempDir("test/fixtures")

func Cleanup() {
	os.RemoveAll(basePath)
}

/*
	Two dirs, of which one contains another file; various permission bits.
	No symlinks or anything fancy.
	Timestamps are varied.
*/
var DirInput1 def.Input

func init() {
	wd := filepath.Join(basePath, "DirInput1")
	DirInput1 = def.Input{
		Type: "dir",
		Hash: "nIf-ikfYp83OWWc_y2D-IGC9WOMYdfMA0l_11TL3VCeFq4QtsU6bBWeXyevujYr4",
		URI:  filepath.Join(wd, "src"),
	}

	if err := os.MkdirAll(filepath.Join(wd, "src"), 0755); err != nil {
		panic(err)
	}
	os.Mkdir(filepath.Join(wd, "src/a"), 01777)
	os.Mkdir(filepath.Join(wd, "src/b"), 0750)
	f, _ := os.OpenFile(filepath.Join(wd, "src/b/c"), os.O_RDWR|os.O_CREATE, 0664)
	f.Write([]byte("zyx"))
	f.Close()
	// since we hash modtimes and this test has a fixture hash, we have to set those up!
	os.Chtimes(filepath.Join(wd, "src"), time.Unix(1, 2), time.Unix(1000, 2000))
	os.Chtimes(filepath.Join(wd, "src/a"), time.Unix(3, 2), time.Unix(3000, 2000))
	os.Chtimes(filepath.Join(wd, "src/b"), time.Unix(5, 2), time.Unix(5000, 2000))
	os.Chtimes(filepath.Join(wd, "src/b/c"), time.Unix(7, 2), time.Unix(7000, 2000))
	// similarly, force uid and gid bits since otherwise they default to your current user, and that's not the same for everyone
	os.Chown(filepath.Join(wd, "src"), 10000, 10000)
	os.Chown(filepath.Join(wd, "src/a"), 10000, 10000)
	os.Chown(filepath.Join(wd, "src/b"), 10000, 10000)
	os.Chown(filepath.Join(wd, "src/b/c"), 10000, 10000)
}

/*
	Basedir with three empty files, no directories.
	Useful for counting to three.
*/
var DirInput2 def.Input

func init() {
	wd := filepath.Join(basePath, "DirInput2")
	DirInput2 = def.Input{
		Type: "dir",
		Hash: "ySN3UeOGyl0E5AU4is6g7V8vUqF8m1PCluq8P23rtBuKT5jDCkTgg7Y65d8DxYbb",
		URI:  filepath.Join(wd, "src"),
	}

	if err := os.MkdirAll(filepath.Join(wd, "src"), 0755); err != nil {
		panic(err)
	}
	ioutil.WriteFile(filepath.Join(wd, "src/1"), []byte{}, 0644)
	ioutil.WriteFile(filepath.Join(wd, "src/2"), []byte{}, 0644)
	ioutil.WriteFile(filepath.Join(wd, "src/3"), []byte{}, 0644)
	// since we hash modtimes and this test has a fixture hash, we have to set those up!
	os.Chtimes(filepath.Join(wd, "src"), time.Unix(1, 2), time.Unix(1000, 2000))
	os.Chtimes(filepath.Join(wd, "src/1"), time.Unix(1, 2), time.Unix(1000, 2000))
	os.Chtimes(filepath.Join(wd, "src/2"), time.Unix(1, 2), time.Unix(1000, 2000))
	os.Chtimes(filepath.Join(wd, "src/3"), time.Unix(1, 2), time.Unix(1000, 2000))
	// similarly, force uid and gid bits since otherwise they default to your current user, and that's not the same for everyone
	os.Chown(filepath.Join(wd, "src"), 10000, 10000)
	os.Chown(filepath.Join(wd, "src/1"), 10000, 10000)
	os.Chown(filepath.Join(wd, "src/2"), 10000, 10000)
	os.Chown(filepath.Join(wd, "src/3"), 10000, 10000)
}
