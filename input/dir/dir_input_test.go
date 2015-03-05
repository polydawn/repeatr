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
					URL:  "/tmp/certainly/should/not/exist",
				})
			}).Catch(def.ValidationError, func(e *errors.Error) {
				correctError = true
			}).Done()

			So(correctError, ShouldBeTrue)
		})
	})

	Convey("Given a directory with a mixture of files and folders", t,
		testutil.WithTmpdir(func() {
			pwd, _ := os.Getwd()
			os.Mkdir("src", 0755)
			os.Mkdir("src/a", 01777)
			os.Mkdir("src/b", 0750)
			f, err := os.OpenFile("src/b/c", os.O_RDWR|os.O_CREATE, 0664)
			So(err, ShouldBeNil)
			f.Write([]byte("zyx"))
			So(f.Close(), ShouldBeNil)

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
					Hash: "abcd",
					URL:  filepath.Join(pwd, "src"),
				})

				Convey("Apply succeeds (hash checks pass)", func() {
					// wait a moment before copying to decrease the odds of nanotime telling a huge, huge lie.
					// time reporting granularity can be extremely arbitrary and without this delay it's possible for the fs timestamps to end up the same before and after copy by pure coincidence, which would make much of the test vacuous.
					// this wait, incidentally, varies in effectiveness with whether you're running with the goconvey web app or working from cli.
					// yes, i'm serious.  i clicked many times to determine this.
					// this is literally the reason why we're building repeatr.
					time.Sleep(2 * time.Millisecond)

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
							SkipSo(fshash.ReadMetadata("dest/b"), ShouldResemble, path2metadata)
							So(fshash.ReadMetadata("dest/b/c"), ShouldResemble, path3metadata) // TODO: needs post-order traversal
							// the top dir should have the same attribs too!  but we have to fix the name.
							destDirMeta := fshash.ReadMetadata("dest/")
							destDirMeta.Name = ""
							SkipSo(destDirMeta, ShouldResemble, path0metadata) // TODO: needs post-order traversal
						})
					})
				})
			})
		}),
	)
}
