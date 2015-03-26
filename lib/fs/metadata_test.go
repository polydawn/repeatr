package fs

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/ugorji/go/codec"
)

func TestMetadataSerialization(t *testing.T) {
	Convey("Given a metadata structure", t, func() {
		metadata := &Metadata{Name: "nom"}
		marshalled := &bytes.Buffer{}
		metadata.Marshal(marshalled)
		marshalled = bytes.NewBuffer(marshalled.Bytes())

		Convey("The marshalled form should be valid cbor", func() {
			dec := codec.NewDecoder(marshalled, new(codec.CborHandle))
			reheated := &Metadata{}
			dec.MustDecode(reheated)
		})

		Convey("The marshalled form should have known keys", func() {
			dec := codec.NewDecoder(marshalled, new(codec.CborHandle))
			reheated := make(map[string]interface{})
			err := dec.Decode(reheated)
			So(err, ShouldBeNil)
			v, exists := reheated["n"]
			So(exists, ShouldBeTrue)
			So(v, ShouldEqual, "nom")
			So(marshalled.Len(), ShouldEqual, 0)
		})
	})
}
