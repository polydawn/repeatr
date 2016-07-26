package def_test

import (
	"sort"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/api/def"
)

func TestMountGroupCodec(t *testing.T) {
	Convey("Given MountGroup", t, func() {
		// initialized in non-canonical order, which is ostensibly Wrong
		//  but we do it to test that serializing still DTRT.
		mg := def.MountGroup{
			def.Mount{
				TargetPath: "/cntrpath/2",
				SourcePath: "/hostpath/same",
				Writable:   true,
			},
			def.Mount{
				TargetPath: "/cntrpath/1",
				SourcePath: "/hostpath/same",
				Writable:   true,
			},
		}

		Convey("Encoding should work", func() {
			buf := encodeToJson(mg)

			Convey("Decoding should bounce back to struct", func() {
				var reheat def.MountGroup
				decodeFromJson(buf.Bytes(), &reheat)
				So(len(reheat), ShouldEqual, 2)
				sort.Sort(def.MountGroupByTargetPath(reheat)) // maybe this just should be an invarient
				So(reheat[0].TargetPath, ShouldEqual, "/cntrpath/1")
				So(reheat[1].TargetPath, ShouldEqual, "/cntrpath/2")
			})

			Convey("Freehand decoding should have expected fields", func() {
				var reheat interface{}
				decodeFromJson(buf.Bytes(), &reheat)
				mp, ok := reheat.(map[interface{}]interface{})
				So(ok, ShouldBeTrue)
				So(len(mp), ShouldEqual, 2)
			})

			Convey("Encoding order should be sorted", func() {
				// Current behavior is to sort by target (aka, container) path.
				// We don't support names right now like input/output groups
				//  simply for lack of a as-yet discovered use case.
				str := buf.String()
				off1 := strings.Index(str, "/1")
				off2 := strings.Index(str, "/2")
				So(off1, ShouldBeLessThan, off2)
			})
		})
	})
}
