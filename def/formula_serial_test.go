package def_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
)

func TestParse(t *testing.T) {
	Convey("Testing formula parse", t, func() {
		placeholderInput := map[string]interface{}{
			"/": map[string]interface{}{
				"type":  "tar",
				"hash":  "abcq",
				"mount": "/",
			},
		}
		placeholderAction := map[string]interface{}{
			"command": "bonk",
		}
		placeholderOutput := map[string]interface{}{
			"/output": map[string]interface{}{
				"type": "tar",
			},
		}

		Convey("Given a basic formula", func() {
			tree := map[string]interface{}{
				"inputs":  placeholderInput,
				"action":  placeholderAction,
				"outputs": placeholderOutput,
			}

			Convey("It should parse", func() {
				formula := &def.Formula{}
				err := formula.Unmarshal(tree)
				So(err, ShouldBeNil)
				So(len(formula.Inputs), ShouldEqual, 1)
				So(formula.Inputs[0].MountPath, ShouldEqual, "/")
				So(formula.Inputs[0].Hash, ShouldEqual, "abcq")
			})
		})

		Convey("Given a formula where mounts are defaulted", func() {
			tree := map[string]interface{}{
				"inputs": map[string]interface{}{
					"/": map[string]interface{}{
						"type": "tar",
						"hash": "abcq",
					},
					"/beep/boop": map[string]interface{}{
						"type": "tar",
						"hash": "abcq",
					},
				},
				"action": placeholderAction,
				"outputs": map[string]interface{}{
					"/beep/boop": map[string]interface{}{
						"type": "tar",
					},
				},
			}

			Convey("The mountpath should be the map key", func() {
				formula := &def.Formula{}
				err := formula.Unmarshal(tree)
				So(err, ShouldBeNil)
				So(len(formula.Inputs), ShouldEqual, 2)
				So(formula.Inputs[0].MountPath, ShouldEqual, "/")
				So(formula.Inputs[1].MountPath, ShouldEqual, "/beep/boop")
				So(len(formula.Outputs), ShouldEqual, 1)
				So(formula.Outputs[0].MountPath, ShouldEqual, "/beep/boop")
			})
		})
	})
}
