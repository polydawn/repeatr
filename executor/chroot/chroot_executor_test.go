package chroot

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	Convey("Given a rootfs input that errors", t,
		testutil.WithTmpdir(func() {
			formula := def.Formula{
				Inputs: []def.Input{
					{
						Type:     "tar",
						Location: "/",
						URI:      "/nonexistance/in/its/most/essential/unform.tar.gz",
					},
				},
			}
			executor := &Executor{
				workspacePath: "chroot_workspace",
			}
			So(os.Mkdir(executor.workspacePath, 0755), ShouldBeNil)

			Convey("We should get an InputError", func() {
				correctError := false
				try.Do(func() {
					_, _ = executor.Run(formula)
				}).Catch(input.InputError, func(e *errors.Error) {
					correctError = true
				}).Done()
				So(correctError, ShouldBeTrue)
			})
		}),
	)

	projPath, _ := os.Getwd()
	projPath = filepath.Dir(filepath.Dir(projPath))

	testutil.Convey_IfHaveRoot("Given a rootfs", t,
		testutil.WithTmpdir(func() {
			formula := def.Formula{
				Inputs: []def.Input{
					{
						Type:     "tar",
						Location: "/",
						URI:      filepath.Join(projPath, "assets/ubuntu.tar.gz"),
					},
				},
			}
			executor := &Executor{
				workspacePath: "chroot_workspace",
			}
			So(os.Mkdir(executor.workspacePath, 0755), ShouldBeNil)

			Convey("The executor should be able to invoke echo", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"echo", "echococo"},
				}

				_, _ = executor.Run(formula)
				// So(job, ShouldNotBeNil)
				// So(outs, ShouldNotBeNil)
				// TODO: spec out how we're going to watch stdout/err from jobs, then test
			})
		}),
	)
}
