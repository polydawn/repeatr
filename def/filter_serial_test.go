package def_test

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
)

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
