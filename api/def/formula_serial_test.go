package def_test

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/api/hitch"
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
			var formula def.Formula
			hitch.DecodeYaml(bytes.NewBuffer(content), &formula)
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
			var formula def.Formula
			hitch.DecodeYaml(bytes.NewBuffer(content), &formula)
			mountsCfg := formula.Action.Escapes.Mounts
			So(len(mountsCfg), ShouldEqual, 1)
			So(mountsCfg[0].SourcePath, ShouldEqual, "/host/files")
			So(mountsCfg[0].TargetPath, ShouldEqual, "/breakout")
		})
	})

	Convey("Given a formula with cradle overrides", t, func() {
		Convey("False is false", func() {
			content := []byte(`
			action:
				cradle: false
			`)
			var formula def.Formula
			hitch.DecodeYaml(bytes.NewBuffer(content), &formula)
			So(formula.Action.Cradle, ShouldNotBeNil)
			So(*formula.Action.Cradle, ShouldEqual, false)
		})
		Convey("True is true", func() {
			content := []byte(`
			action:
				cradle: true
			`)
			var formula def.Formula
			hitch.DecodeYaml(bytes.NewBuffer(content), &formula)
			So(formula.Action.Cradle, ShouldNotBeNil)
			So(*formula.Action.Cradle, ShouldEqual, true)
		})
		Convey("Absense is nil", func() {
			content := []byte(``)
			var formula def.Formula
			hitch.DecodeYaml(bytes.NewBuffer(content), &formula)
			So(formula.Action.Cradle, ShouldBeNil)
		})
	})

	Convey("Given a formula with policy settings", t, func() {
		Convey("Valid enum values parse", func() {
			content := []byte(`
			action:
				policy: governor
			`)
			var formula def.Formula
			hitch.DecodeYaml(bytes.NewBuffer(content), &formula)
			So(formula.Action.Policy, ShouldEqual, def.PolicyGovernor)
		})
		Convey("Non-enum values should be rejected", func() {
			content := []byte(`
			action:
				policy: nonsense
			`)
			var formula def.Formula
			So(func() {
				hitch.DecodeYaml(bytes.NewBuffer(content), &formula)
			}, ShouldPanic)
		})
	})
}
