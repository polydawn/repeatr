package cereal

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParse(t *testing.T) {
	Convey("Testing string normalization", t, func() {
		var str string
		str += "wat:\n"
		str += "\tbat:\n"
		str += "\t\tnat:\n"

		Convey("tab2space dtrt", func() {
			So(string(Tab2space([]byte(str))), ShouldEqual, strings.Replace(str, "\t", "  ", -1))
		})
	})
}
