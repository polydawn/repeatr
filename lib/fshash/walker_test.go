package fshash

import (
	"crypto/sha512"
	"os"
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	Convey("Given a directory with a mixture of files and folders", t,
		testutil.WithTmpdir(func() {
			os.Mkdir("src", 0755)
			os.Mkdir("src/a", 01777)
			os.Mkdir("src/b", 0750)
			f, err := os.OpenFile("src/b/c", os.O_RDWR|os.O_CREATE, 0664)
			So(err, ShouldBeNil)
			f.Write([]byte("zyx"))
			So(f.Close(), ShouldBeNil)

			Convey("We can walk and fill a bucket", func() {
				bucket := &MemoryBucket{}
				err := FillBucket("src", "", bucket, sha512.New384)
				So(err, ShouldBeNil)

				Convey("Then the bucket contains the file descriptions", func() {
					So(len(bucket.lines), ShouldEqual, 4)
					sort.Sort(linesByFilepath(bucket.lines))
					So(bucket.lines[0].metadata.Name, ShouldEqual, ".")
					So(bucket.lines[1].metadata.Name, ShouldEqual, "./a")
					So(bucket.lines[2].metadata.Name, ShouldEqual, "./b")
					So(bucket.lines[3].metadata.Name, ShouldEqual, "./b/c")
				})

				Convey("Doing it again produces identical descriptions", func() {
					bucket2 := &MemoryBucket{}
					err := FillBucket("src", "", bucket2, sha512.New384)
					So(err, ShouldBeNil)

					So(len(bucket2.lines), ShouldEqual, 4)
					sort.Sort(linesByFilepath(bucket.lines))
					sort.Sort(linesByFilepath(bucket2.lines))
					for i := range bucket.lines {
						So(bucket2.lines[i], ShouldResemble, bucket.lines[i])
					}
				})
			})
		}),
	)
}
