package streamer

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/ugorji/go/codec"
	"polydawn.net/repeatr/testutil"
)

func TestCborMux(t *testing.T) {
	Convey("Using a cbor file-backed streamer mux", t, testutil.WithTmpdir(func() {
		strm := CborFileMux("./logfile")

		Convey("Given a single complete stream", func() {
			a1 := strm.Appender(1)
			a1.Write([]byte("asdf"))
			a1.Write([]byte("qwer"))
			a1.Close()

			Convey("Readall should get the whole stream", func() {
				r1 := strm.Reader(1)
				bytes, err := ioutil.ReadAll(r1)
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, "asdfqwer")

				Convey("Readall *again* should get the whole stream, from the beginning", func() {
					r1 := strm.Reader(1)
					bytes, err := ioutil.ReadAll(r1)
					So(err, ShouldBeNil)
					So(string(bytes), ShouldEqual, "asdfqwer")
				})
			})
		})

		Convey("Given two complete streams", func() {
			a1 := strm.Appender(1)
			a2 := strm.Appender(2)
			a1.Write([]byte("asdf"))
			a2.Write([]byte("qwer"))
			a1.Write([]byte("asdf"))
			a1.Close()
			a2.Write([]byte("zxcv"))
			a2.Close()

			Convey("Readall on one label should get the whole stream for that label", func() {
				r1 := strm.Reader(1)
				bytes, err := ioutil.ReadAll(r1)
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, "asdfasdf")
			})
			Convey("Readall on both labels should get the whole stream", func() {
				r12 := strm.Reader(1, 2)
				bytes, err := ioutil.ReadAll(r12)
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, "asdfqwerasdfzxcv")
			})
			// TODO read it back manually with tiny buffers
		})

		// TODO still need a zillion tests around EOF and blocking behavior near that.

		Convey("It should parse as regular cbor", func() {
			a1 := strm.Appender(1)
			a1.Write([]byte("asdf"))
			a2 := strm.Appender(2)
			a2.Write([]byte("qwer"))
			a1.Close()
			a2.Close()
			strm.(*CborMux).Close()

			file, err := os.OpenFile("./logfile", os.O_RDONLY, 0)
			So(err, ShouldBeNil)
			dec := codec.NewDecoder(file, new(codec.CborHandle))
			reheated := make([]cborMuxRow, 0)
			dec.MustDecode(&reheated)
			So(reheated, ShouldResemble, []cborMuxRow{
				{Label: 1, Msg: []byte("asdf")},
				{Label: 2, Msg: []byte("qwer")},
			})
		})
	}))
}
