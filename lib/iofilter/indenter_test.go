package iofilter

import (
	"bytes"
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test(t *testing.T) {
	Convey("Indenting writers should DTRT", t, FailureContinues, func() {
		Convey("Given a single line", func() {
			buf := bytes.NewBufferString("msg\n")
			var out bytes.Buffer
			n, err := io.Copy(LineIndentingWriter(&out), buf)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4)
			So(out.String(), ShouldResemble, "\tmsg\n")
		})

		Convey("Given a couple lines", func() {
			buf := bytes.NewBufferString("\nwow\ndang\nmsg\n")
			var out bytes.Buffer
			n, err := io.Copy(LineIndentingWriter(&out), buf)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 1+3+1+4+1+3+1)
			So(out.String(), ShouldResemble, "\t\n\twow\n\tdang\n\tmsg\n")
		})

		Convey("Unterminated lines don't get flushed, sorry", func() {
			buf := bytes.NewBufferString("msg1\nmsg2")
			var out bytes.Buffer
			n, err := io.Copy(LineIndentingWriter(&out), buf)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, 4+1+4)
			So(out.String(), ShouldResemble, "\tmsg1\n")
		})

		Convey("Nested indenters work", func() {
			var out bytes.Buffer
			wr := LineIndentingWriter(&out)
			wr.Write([]byte("msg1\n"))
			LineIndentingWriter(wr).Write([]byte("msg2\n"))
			wr.Write([]byte("msg3\n"))
			So(out.String(), ShouldResemble, "\tmsg1\n\t\tmsg2\n\tmsg3\n")
		})
	})
}
