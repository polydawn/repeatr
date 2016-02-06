package catalog

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCatalogs(t *testing.T) {
	Convey("Catalogs should be immutable", t, func(c C) {
		cat := New(ID("catname"))
		catEd1 := cat.Release("", SKU{Hash: "thing1"})
		catEd2 := catEd1.Release("", SKU{Hash: "thing2"})

		Convey("Releases give a bigger catalog", func() {
			So(catEd2.Tracks[""], ShouldHaveLength, 2)

			Convey("Releases don't mutate existing catalog references", func() {
				So(catEd1.Tracks[""], ShouldHaveLength, 1)
			})
		})
	})
}
