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

		Convey("It should be transparent with a single stream", func() {
			a1 := strm.Appender(1)
			a1.Write([]byte("asdf"))
			a1.Write([]byte("qwer"))

			r1 := strm.Reader(1)
			bytes, err := ioutil.ReadAll(r1)
			So(err, ShouldBeNil)
			So(string(bytes), ShouldEqual, "asdfqwer")
		})

		Convey("It should be transparent with two streams", func() {

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
			dec := codec.NewDecoder(&debugReader{file}, new(codec.CborHandle))
			reheated := make([]interface{}, 0)
			dec.MustDecode(&reheated)
			SkipSo(reheated, ShouldResemble, []interface{}{
				map[interface{}]interface{}{
					byte(1): []byte("asdf"),
				},
				map[interface{}]interface{}{
					byte(2): []byte("qwer"),
				},
			})
		})
	}))
}
