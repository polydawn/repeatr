package def_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	rdef "go.polydawn.net/repeatr/api/def"
)

func TestFormulaHashFixtures(t *testing.T) {
	Convey("Formulas should hash consistently", t, func() {
		Convey("Given fixture 1", func() {
			frm := &rdef.Formula{
				Inputs: rdef.InputGroup{
					"rootfs": &rdef.Input{
						Type:       "tar",
						Hash:       "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT",
						Warehouses: rdef.WarehouseCoords{"http+ca://repeatr.s3.amazonaws.com/assets/"},
						MountPath:  "/",
					},
				},
				Action: rdef.Action{
					Entrypoint: []string{"bash", "-c", "echo hello && sleep 5 && echo yes"},
				},
			}
			Convey("It should match the fixture", func() {
				So(frm.Hash(), ShouldEqual, "7dCwntFg7FtcUNs4FcRaUknXWWskG889kypU9y3BpWdVmT4aMA76zkZyjaNYL1x989")
			})
			Convey("Given a fixture that varies only in warehouses, it should match the same fixture", func() {
				frm.Inputs["rootfs"].Warehouses = rdef.WarehouseCoords{"file+ca://./local/"}
				So(frm.Hash(), ShouldEqual, "7dCwntFg7FtcUNs4FcRaUknXWWskG889kypU9y3BpWdVmT4aMA76zkZyjaNYL1x989")
			})
			Convey("Given changes in outputs, it should have a different fixture", func() {
				frm.Outputs = rdef.OutputGroup{
					"product": &rdef.Output{
						Type:      "tar",
						MountPath: "/output",
					},
				}
				So(frm.Hash(), ShouldEqual, "8h8bRDwDAS39QtyQ7SNn9BYKZkXCxkxWMpCcHAMKdbvYe2qQ3r7TduVccHeCe8dk4y")
			})
			Convey("Given non-conjectured outputs with hashes, it should match the same fixture", func() {
				frm.Outputs = rdef.OutputGroup{
					"product": &rdef.Output{
						Type:       "tar",
						MountPath:  "/output",
						Conjecture: false,
						Hash:       "baby's first hash",
					},
				}
				So(frm.Hash(), ShouldEqual, "8h8bRDwDAS39QtyQ7SNn9BYKZkXCxkxWMpCcHAMKdbvYe2qQ3r7TduVccHeCe8dk4y")
			})
			Convey("Given conjectured outputs with hashes, it should have a different fixture", func() {
				frm.Outputs = rdef.OutputGroup{
					"product": &rdef.Output{
						Type:       "tar",
						MountPath:  "/output",
						Conjecture: true,
						Hash:       "baby's first hash",
					},
				}
				So(frm.Hash(), ShouldEqual, "7rViXY4NfCiveRz3D5scyTJnJd9YegRgvwtiLFGXGyn6bzqM8H2wGxhmQw3bgvcnsp")
			})
		})
	})
}
