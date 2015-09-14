package def_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
)

func TestParse(t *testing.T) {
	Convey("Testing formula parse", t, func() {
		tree := map[string]interface{}{
			"inputs": map[string]interface{}{
				"/": map[string]interface{}{
					"type":  "tar",
					"hash":  "abcq",
					"mount": "/",
				},
			},
			"action": map[string]interface{}{
				"command": "bonk",
			},
			"outputs": map[string]interface{}{
				"/output": map[string]interface{}{
					"type": "tar",
				},
			},
		}
		formula := &def.Formula{}
		err := formula.Unmarshal(tree)
		So(err, ShouldBeNil)
		So(len(formula.Inputs), ShouldEqual, 1)
		So(formula.Inputs[0].MountPath, ShouldEqual, "/")
		So(formula.Inputs[0].Hash, ShouldEqual, "abcq")
	})
}
