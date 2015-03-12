package dir

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	Convey("Given a nonexistant path", t, func() {
		Convey("The input config should be rejected during validation", func() {
			correctError := false
			try.Do(func() {
				New(def.Input{
					Type: "dir",
					Hash: "abcd",
					URI:  "/tmp/certainly/should/not/exist",
				})
			}).Catch(def.ValidationError, func(e *errors.Error) {
				correctError = true
			}).Done()

			So(correctError, ShouldBeTrue)
		})
	})

	testutil.Convey_IfHaveRoot("Given a directory with a mixture of files and folders", t,
		testutil.WithTmpdir(func() {
			pwd, _ := os.Getwd()
			os.Mkdir("src", 0755)
			os.Mkdir("src/a", 01777)
			os.Mkdir("src/b", 0750)
			f, err := os.OpenFile("src/b/c", os.O_RDWR|os.O_CREATE, 0664)
			So(err, ShouldBeNil)
			f.Write([]byte("zyx"))
			So(f.Close(), ShouldBeNil)

			// since we hash modtimes and this test has a fixture hash, we have to set those up!
			So(os.Chtimes("src", time.Unix(1, 2), time.Unix(1000, 2000)), ShouldBeNil)
			So(os.Chtimes("src/a", time.Unix(3, 2), time.Unix(3000, 2000)), ShouldBeNil)
			So(os.Chtimes("src/b", time.Unix(5, 2), time.Unix(5000, 2000)), ShouldBeNil)
			So(os.Chtimes("src/b/c", time.Unix(7, 2), time.Unix(7000, 2000)), ShouldBeNil)
			// similarly, force uid and gid bits since otherwise they default to your current user, and that's not the same for everyone
			So(os.Chown("src", 10000, 10000), ShouldBeNil)
			So(os.Chown("src/a", 10000, 10000), ShouldBeNil)
			So(os.Chown("src/b", 10000, 10000), ShouldBeNil)
			So(os.Chown("src/b/c", 10000, 10000), ShouldBeNil)

			fixtureHash := "nIf-ikfYp83OWWc_y2D-IGC9WOMYdfMA0l_11TL3VCeFq4QtsU6bBWeXyevujYr4"

			// save attributes first because access times are conceptually insane
			// remarkably, since the first read doesn't cause atimes to change,
			// the inputter can capture it and we can recreate it.
			// but that still doesn't make anything else about checking or handling it sane.
			path0metadata := fshash.ReadMetadata("src")
			path0metadata.Name = ""
			path1metadata := fshash.ReadMetadata("src/a")
			path2metadata := fshash.ReadMetadata("src/b")
			path3metadata := fshash.ReadMetadata("src/b/c")

			Convey("We can construct an input", func() {
				inputter := New(def.Input{
					Type: "dir",
					Hash: fixtureHash,
					URI:  filepath.Join(pwd, "src"),
				})

				Convey("Apply succeeds (hash fixture checks pass)", func() {
					waitCh := inputter.Apply(filepath.Join(pwd, "dest"))
					So(<-waitCh, ShouldBeNil)

					Convey("The destination files exist", func() {
						So("dest/a", testutil.ShouldBeFile, os.ModeDir)
						So("dest/b", testutil.ShouldBeFile, os.ModeDir)
						So("dest/b/c", testutil.ShouldBeFile, os.FileMode(0))
						content, err := ioutil.ReadFile("dest/b/c")
						So(err, ShouldBeNil)
						So(string(content), ShouldEqual, "zyx")

						Convey("And all metadata matches", func() {
							// Comparing fileinfo doesn't work conveniently; you keep getting new pointers for 'sys'
							//one, _ := os.Lstat("src/a")
							//two, _ := os.Lstat("dest/a")
							//So(one, ShouldResemble, two)
							So(fshash.ReadMetadata("dest/a"), ShouldResemble, path1metadata)
							So(fshash.ReadMetadata("dest/b"), ShouldResemble, path2metadata)
							So(fshash.ReadMetadata("dest/b/c"), ShouldResemble, path3metadata)
							// the top dir should have the same attribs too!  but we have to fix the name.
							destDirMeta := fshash.ReadMetadata("dest/")
							destDirMeta.Name = ""
							So(destDirMeta, ShouldResemble, path0metadata)
						})
					})

					Convey("Copying the copy should still match on hash", func() {
						inputter2 := New(def.Input{
							Type: "dir",
							Hash: fixtureHash,
							URI:  filepath.Join(pwd, "dest"),
						})

						waitCh := inputter2.Apply(filepath.Join(pwd, "copycopy"))
						So(<-waitCh, ShouldBeNil)
					})
				})
			})

			Convey("A different hash is rejected", func() {
				inputter := New(def.Input{
					Type: "dir",
					Hash: "abcd",
					URI:  filepath.Join(pwd, "src"),
				})
				err := <-inputter.Apply(filepath.Join(pwd, "dest"))
				So(err, testutil.ShouldBeErrorClass, input.InputHashMismatchError)
			})

			Convey("A change in content breaks the hash", func() {
				// we could do separate tests for added and removed, but those don't trigger markedly different paths so i think we're pretty well covered already.
				inputter := New(def.Input{
					Type: "dir",
					Hash: fixtureHash,
					URI:  filepath.Join(pwd, "src"),
				})
				f, err := os.OpenFile("src/b/c", os.O_RDWR, 0664)
				So(err, ShouldBeNil)
				f.Write([]byte("222"))
				So(f.Close(), ShouldBeNil)
				err = <-inputter.Apply(filepath.Join(pwd, "dest"))
				So(err, testutil.ShouldBeErrorClass, input.InputHashMismatchError)
			})
		}),
	)
}
