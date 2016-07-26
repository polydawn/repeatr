package cradle

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/lib/testutil"
)

func TestCradleFilesystem(t *testing.T) {
	Convey("Given a blank rootfs", t, testutil.WithTmpdir(func(c C) {
		Convey("ensureTempDir works", func() {
			ensureTempDir("./")
			So("./tmp", testutil.ShouldBeFile, os.FileMode(0777)|os.ModeSticky|os.ModeDir)
		})
	}))

	Convey("Given a mostly complete but very strange rootfs", t, testutil.WithTmpdir(func(c C) {
		os.Mkdir("./tmp", 0755)
		Convey("ensureTempDir resets perms", func() {
			ensureTempDir("./")
			So("./tmp", testutil.ShouldBeFile, os.FileMode(0777)|os.ModeSticky|os.ModeDir)
		})
	}))
}
