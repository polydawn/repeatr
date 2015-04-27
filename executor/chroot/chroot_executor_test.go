package chroot

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/input/fixtures"
	"polydawn.net/repeatr/lib/guid"
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
				result := e.Start(formula, def.JobID(guid.New()), ioutil.Discard).Wait()
				So(result.Error, testutil.ShouldBeErrorClass, input.Error)
			})

			Convey("The job exit code should clearly indicate failure", FailureContinues, func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"echo", "echococo"},
				}
				job := e.Start(formula, def.JobID(guid.New()), ioutil.Discard)
				So(job, ShouldNotBeNil)
				So(job.Wait().Error, ShouldNotBeNil)
				// Even though one should clearly also check the error status,
				//  zero here could be very confusing, so jobs that error before start should be -1.
				So(job.Wait().ExitCode, ShouldEqual, -1)
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
						Hash:     "b6nXWuXamKB3TfjdzUSL82Gg1avuvTk0mWQP4wgegscZ_ZzG9GfHDwKXQ9BfCx6v",
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

				job := e.Start(formula, def.JobID(guid.New()), ioutil.Discard)
				So(job, ShouldNotBeNil)
				// note that we can read output concurrently.
				// no need to wait for job done.
				msg, err := ioutil.ReadAll(job.OutputReader())
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "echococo\n")
				So(job.Wait().Error, ShouldBeNil)
				So(job.Wait().ExitCode, ShouldEqual, 0)
			})

			Convey("The executor should be able to check exit codes", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"sh", "-c", "exit 14"},
				}

				job := e.Start(formula, def.JobID(guid.New()), ioutil.Discard)
				So(job, ShouldNotBeNil)
				So(job.Wait().Error, ShouldBeNil)
				So(job.Wait().ExitCode, ShouldEqual, 14)
			})

			Convey("The executor should report command not found clearly", func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"not a command"},
				}

				result := e.Start(formula, def.JobID(guid.New()), ioutil.Discard).Wait()
				So(result.Error, testutil.ShouldBeErrorClass, executor.NoSuchCommandError)
			})

			Convey("Given another input", func() {
				inputfixtures.DirInput2.Location = "/data/test"
				formula.Inputs = append(formula.Inputs, inputfixtures.DirInput2)

				Convey("The executor should be able to see the mounted files", FailureContinues, func() {
					formula.Accents = def.Accents{
						Entrypoint: []string{"ls", "/data/test"},
					}

					job := e.Start(formula, def.JobID(guid.New()), ioutil.Discard)
					So(job, ShouldNotBeNil)
					So(job.Wait().Error, ShouldBeNil)
					So(job.Wait().ExitCode, ShouldEqual, 0)
					msg, err := ioutil.ReadAll(job.OutputReader())
					So(err, ShouldBeNil)
					So(string(msg), ShouldEqual, "1\n2\n3\n")
				})
			})
		}),
	)
}
