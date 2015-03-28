package chroot

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/input/fixtures"
	"polydawn.net/repeatr/input/tar"
	"polydawn.net/repeatr/testutil"
)

func TestMain(m *testing.M) {
	code := m.Run()
	inputfixtures.Cleanup()
	os.Exit(code)
}

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
			e := &Executor{
				workspacePath: "chroot_workspace",
			}
			So(os.Mkdir(e.workspacePath, 0755), ShouldBeNil)

			Convey("We should get an InputError", func() {
				result := e.Start(formula).Wait()
				So(result.Error, testutil.ShouldBeErrorClass, tar.Error)
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
			e := &Executor{
				workspacePath: "chroot_workspace",
			}
			So(os.Mkdir(e.workspacePath, 0755), ShouldBeNil)

			Convey("The executor should be able to invoke echo", FailureContinues, func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"echo", "echococo"},
				}

				job := e.Start(formula)
				So(job, ShouldNotBeNil)
				So(job.Wait().ExitCode, ShouldEqual, 0) // TODO: this waits... the test should still pass if the reader happens first
			})

			Convey("The executor should be able to check exit codes", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"sh", "-c", "exit 14"},
				}

				job := e.Start(formula)
				So(job, ShouldNotBeNil)
				So(job.Wait().ExitCode, ShouldEqual, 14)
			})

			Convey("The executor should report command not found clearly", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"not a command"},
				}

				result := e.Start(formula).Wait()
				So(result.Error, testutil.ShouldBeErrorClass, executor.NoSuchCommandError)
			})

			Convey("Given another input", func() {
				inputfixtures.DirInput2.Location = "/data/test"
				formula.Inputs = append(formula.Inputs, inputfixtures.DirInput2)

				Convey("The executor should be able to see the mounted files", FailureContinues, func() {
					formula.Accents = def.Accents{
						Entrypoint: []string{"ls", "/data/test"},
					}

					job := e.Start(formula)
					So(job, ShouldNotBeNil)
					So(job.Wait().ExitCode, ShouldEqual, 0)
				})
			})
		}),
	)
}
