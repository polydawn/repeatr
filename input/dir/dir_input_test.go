package dir

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/testutil"
)

func TestThings(t *testing.T) {

}

func Test(t *testing.T) {
	Convey("Given a nonexistant path", t, func() {
		Convey("The input config should be rejected during validation", func() {
			correctError := false
			try.Do(func() {
				New(def.Input{
					Type:     "dir",
					Hash:     "abcd",
					URL:      "/tmp/certainly/should/not/exist",
					Location: "/data",
					// REVIEW: two of these four fields were more for the executor.
					// maybe this struct isn't the sanest to use for this arg.
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

			Convey("We can construct an input", func() {
				inputter := New(def.Input{
					Type: "dir",
					Hash: "abcd",
					URL:  filepath.Join(pwd, "src"),
				})

				Convey("Apply succeeds (hash checks pass)", func() {
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

						})
					})
				})
			})
		}),
	)
}
