package def_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
)

func TestStringParse(t *testing.T) {
	Convey("Given a basic formula", t, func() {
		content := []byte(`
		inputs:
			"/":
				type: "bonk"
				hash: "asdf"
		action:
			command:
				- "shellit"
		`)

		Convey("It should parse", func() {
			formula := def.ParseYaml(content)
			So(len(formula.Inputs), ShouldEqual, 1)
			So(formula.Inputs["/"].MountPath, ShouldEqual, "/")
			So(formula.Inputs["/"].Hash, ShouldEqual, "asdf")
		})
	})

	Convey("Given a formula with mount escapes", t, func() {
		content := []byte(`
		inputs:
			"/":
				type: "bonk"
				hash: "asdf"
		action:
			command:
				- "shellit"
			escapes:
				mounts:
					"/breakout": "/host/files"
		outputs:
			"/dev/null":
				type: "nope"
		`)

		Convey("It should parse", func() {
			formula := def.ParseYaml(content)
			mountsCfg := formula.Action.Escapes.Mounts
			So(len(mountsCfg), ShouldEqual, 1)
			So(mountsCfg[0].SourcePath, ShouldEqual, "/host/files")
			So(mountsCfg[0].TargetPath, ShouldEqual, "/breakout")
		})
	})
}
