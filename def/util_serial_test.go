package def_test

import (
	"bytes"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/ugorji/go/codec"
)

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
