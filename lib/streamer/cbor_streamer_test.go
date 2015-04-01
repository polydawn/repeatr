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

			r1 := strm.Reader(1)
			bytes, err := ioutil.ReadAll(r1)
			So(err, ShouldBeNil)
			So(string(bytes), ShouldEqual, "asdfqwer")
		})

		Convey("Given two complete streams", func() {

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
