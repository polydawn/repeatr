package def_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/api/def"
)

func TestRunRecordCodec(t *testing.T) {
	Convey("Given a RunRecord containing failure info", t, func() {
		rr := def.RunRecord{
			UID: "whee",
			Failure: &def.ErrWareDNE{
				Ware: def.Ware{"fmt", "asdf"},
				From: "url",
			},
		}

		Convey("The json should match fixtures", func() {
			ser := encodeToJson(rr).String()
			So(ser, ShouldEqual,
				`{"UID":"whee","failure":{"detail":{"from":"url","ware":{"hash":"asdf","type":"fmt"}},"type":"ErrWareDNE"},"results":null,"when":"0001-01-01T00:00:00Z"}`)
		})

		Convey("The bounce should be equal", func() {
			buf := encodeToJson(rr)
			var rr2 def.RunRecord
			decodeFromJson(buf.Bytes(), &rr2)
			So(rr2.UID, ShouldResemble, rr.UID)
			So(rr2.Failure, ShouldResemble, rr.Failure)
		})
	})
}
