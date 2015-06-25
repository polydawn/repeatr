package fshash

import (
	"crypto/sha512"
	"os"
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/io/filter"
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
			f, err = os.OpenFile("src/d", os.O_RDWR|os.O_CREATE, 0755)
			So(err, ShouldBeNil)
			f.Write([]byte("jkl"))
			So(f.Close(), ShouldBeNil)

			Convey("We can walk and fill a bucket", func() {
				bucket := &MemoryBucket{}
				err := FillBucket("src", "", bucket, filter.FilterSet{}, sha512.New384)
				So(err, ShouldBeNil)

				Convey("Then the bucket contains the file descriptions", func() {
					So(len(bucket.lines), ShouldEqual, 5)
					sort.Sort(linesByFilepath(bucket.lines))
					So(bucket.lines[0].Metadata.Name, ShouldEqual, "./")
					So(bucket.lines[1].Metadata.Name, ShouldEqual, "./a/")
					So(bucket.lines[2].Metadata.Name, ShouldEqual, "./b/")
					So(bucket.lines[3].Metadata.Name, ShouldEqual, "./b/c")
					So(bucket.lines[4].Metadata.Name, ShouldEqual, "./d")
				})

				Convey("Doing it again produces identical descriptions", func() {
					bucket2 := &MemoryBucket{}
					err := FillBucket("src", "", bucket2, filter.FilterSet{}, sha512.New384)
					So(err, ShouldBeNil)

					So(len(bucket2.lines), ShouldEqual, 5)
					sort.Sort(linesByFilepath(bucket.lines))
					sort.Sort(linesByFilepath(bucket2.lines))
					for i := range bucket.lines {
						So(bucket2.lines[i], ShouldResemble, bucket.lines[i])
					}
				})

				Convey("We can walk the bucket and touch all records", func() {
					root := bucket.Iterator()
					So(root.Record().Metadata.Name, ShouldEqual, "./")
					node_a := root.NextChild().(RecordIterator)
					So(node_a.Record().Metadata.Name, ShouldEqual, "./a/")
					So(node_a.NextChild(), ShouldBeNil)
					node_b := root.NextChild().(RecordIterator)
					So(node_b.Record().Metadata.Name, ShouldEqual, "./b/")
					node_c := node_b.NextChild().(RecordIterator)
					So(node_c.Record().Metadata.Name, ShouldEqual, "./b/c")
					So(node_c.NextChild(), ShouldBeNil)
					node_d := root.NextChild().(RecordIterator)
					So(node_d.Record().Metadata.Name, ShouldEqual, "./d")
					So(node_d.NextChild(), ShouldBeNil)
					So(root.NextChild(), ShouldBeNil)
				})
			})

			Convey("We can walk and make a copy while filling a bucket", func() {
				bucket := &MemoryBucket{}
				err := FillBucket("src", "dest", bucket, filter.FilterSet{}, sha512.New384)
				So(err, ShouldBeNil)

				Convey("Walking the copy should match on hash", func() {
					bucket2 := &MemoryBucket{}
					err := FillBucket("dest", "", bucket2, filter.FilterSet{}, sha512.New384)
					So(err, ShouldBeNil)

					hash1 := Hash(bucket, sha512.New384)
					hash2 := Hash(bucket2, sha512.New384)
					So(hash2, ShouldResemble, hash1)
				})
			})
		}),
	)
}
