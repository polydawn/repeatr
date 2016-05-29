package def_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/api/def"
)

func TestInputGroupCodec(t *testing.T) {
	Convey("Given an InputGroup map", t, func() {
		// this is unordered
		ig := def.InputGroup{
			"betic": &def.Input{MountPath: "/2"},
			"alpha": &def.Input{MountPath: "/1"},
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

func TestFilterCodec(t *testing.T) {
	Convey("Given some partially configured Filters", t, func() {
		filt := def.Filters{
			GidMode: def.FilterUse,
		}

		Convey("Encoding should work", func() {
			buf := encodeToJson(filt)

			Convey("Decoding should bounce back to struct", func() {
				var reheat def.Filters
				decodeFromJson(buf.Bytes(), &reheat)
				So(reheat, ShouldResemble, filt)
			})

			Convey("Encoding should match fixture", func() {
				So(buf.String(), ShouldResemble,
					`["gid 0"]`)
			})
		})
	})
	Convey("Given filter strings with dates", t, func() {
		filtStrs := `["gid 12", "mtime 2006-01-02T15:04:05+07:00"]`

		Convey("Decoding should work", func() {
			var filt def.Filters
			decodeFromJson([]byte(filtStrs), &filt)

			Convey("Decoding match expected struct", func() {
				So(filt, ShouldResemble, def.Filters{
					GidMode:   def.FilterUse,
					Gid:       12,
					MtimeMode: def.FilterUse,
					Mtime:     time.Unix(1136189045, 0).UTC(),
				})
			})

			Convey("Reencoding should match, but with unix-time format", func() {
				So(encodeToJson(filt).String(), ShouldResemble,
					`["gid 12","mtime @1136189045"]`)
			})
		})
	})
}
