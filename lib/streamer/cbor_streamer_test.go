package streamer

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/ugorji/go/codec"
	"go.polydawn.net/repeatr/lib/testutil"
)

type resp struct {
	msg []byte
	err error
}

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

			Convey("Small reads shouldn't lose parts", func() {
				// small reads should:
				// 1. finish any previous chunks if buffered from a prior small read -- and then return, without starting a new chunk
				// 2. return as much as they can
				r1 := strm.Reader(1)
				buf := make([]byte, 3)
				n, err := r1.Read(buf)
				So(err, ShouldBeNil)
				So(n, ShouldEqual, 3)
				So(string(buf[:n]), ShouldEqual, "asd")
				n, err = r1.Read(buf)
				So(err, ShouldBeNil)
				So(n, ShouldEqual, 1)
				So(string(buf[:n]), ShouldEqual, "f")

				// make the buffer even more small, so it takes >2 reads
				buf = make([]byte, 1)
				n, err = r1.Read(buf)
				So(err, ShouldBeNil)
				So(n, ShouldEqual, 1)
				So(string(buf[:n]), ShouldEqual, "q")
				n, err = r1.Read(buf)
				So(err, ShouldBeNil)
				So(n, ShouldEqual, 1)
				So(string(buf[:n]), ShouldEqual, "w")
				n, err = r1.Read(buf)
				So(err, ShouldBeNil)
				So(n, ShouldEqual, 1)
				So(string(buf[:n]), ShouldEqual, "e")
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
		})

		Convey("Given two in-progress streams", func() {
			a1 := strm.Appender(1)
			a2 := strm.Appender(2)
			a1.Write([]byte("asdf"))
			a2.Write([]byte("qwer"))

			Convey("Readall on one label should not return yet", FailureContinues, func() {
				r1 := strm.Reader(1)
				r1chan := make(chan resp)
				go func() {
					bytes, err := ioutil.ReadAll(r1)
					r1chan <- resp{bytes, err}
				}()
				select {
				case <-r1chan:
					So(true, ShouldBeFalse)
				default:
					// should be blocked and bounce out here
					So(true, ShouldBeTrue)
				}

				Convey("Sending more bytes and closing should be readable", func() {
					a1.Write([]byte("zxcv"))
					select {
					case <-r1chan:
						So(true, ShouldBeFalse)
					default:
						// should be blocked and bounce out here
						So(true, ShouldBeTrue)
					}

					a1.Close()
					select {
					case resp := <-r1chan:
						So(resp.err, ShouldBeNil)
						So(string(resp.msg), ShouldEqual, "asdfzxcv")
					case <-time.After(1 * time.Second):
						So(true, ShouldBeFalse)
					}
				})
			})
		})

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
				{Label: 1, Sig: 1},
				{Label: 2, Sig: 1},
			})
		})
	}))
}
