package def_test

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/def"
)

func TestInputGroupCodec(t *testing.T) {
	Convey("Given an InputGroup map", t, func() {
		// this is unordered
		ig := def.InputGroup{
			"betic": &def.Input{Name: "/2"},
			"alpha": &def.Input{Name: "/1"},
		}

		Convey("Encoding should work", func() {
			buf := encodeToJson(ig)

			Convey("Decoding should bounce back to struct", func() {
				var reheat def.InputGroup
				decodeFromJson(buf.Bytes(), &reheat)
				So(len(reheat), ShouldEqual, 2)
				So(reheat["alpha"], ShouldResemble, ig["alpha"])
				So(reheat["betic"], ShouldResemble, ig["betic"])
			})

			Convey("Freehand decoding should have expected fields", func() {
				var reheat interface{}
				decodeFromJson(buf.Bytes(), &reheat)
				mp, ok := reheat.(map[interface{}]interface{})
				So(ok, ShouldBeTrue)
				So(len(mp), ShouldEqual, 2)
			})

			Convey("Encoding order should be sorted", func() {
				// I can't think of a better way to check this, since any deserialization necessary snuffles it again
				str := buf.String()
				off1 := strings.Index(str, "alpha")
				off2 := strings.Index(str, "betic")
				So(off1, ShouldBeLessThan, off2)
			})
		})
	})
}

func encodeToJson(x interface{}) *bytes.Buffer {
	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(x)
	So(err, ShouldBeNil)
	return buf
}

func decodeFromJson(b []byte, x interface{}) {
	err := codec.NewDecoderBytes(b, &codec.JsonHandle{}).Decode(&x)
	So(err, ShouldBeNil)
}
