package chroot

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/tests"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/testutil"
	"polydawn.net/repeatr/testutil/filefixture"
)

func Test(t *testing.T) {
	Convey("Spec Compliance: Chroot Executor", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				execEng := &Executor{
					workspacePath: "chroot_workspace",
				}
				So(os.Mkdir(execEng.workspacePath, 0755), ShouldBeNil)

				tests.CheckBasicExecution(execEng)
			}),
		),
	)

	projPath, _ := os.Getwd()
	projPath = filepath.Dir(filepath.Dir(projPath))

	Convey("Given a rootfs", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				formula := def.Formula{
					Inputs: []def.Input{
						{
							Type:     "tar",
							Location: "/",
							Hash:     "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL",
							URI:      "file://" + filepath.Join(projPath, "assets/ubuntu.tar.gz"),
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
					filefixture.Beta.Create("./fixture/beta")
					formula.Inputs = append(formula.Inputs, (def.Input{
						Type:     "dir",
						Hash:     filefixture.Beta_Hash,
						URI:      "./fixture/beta",
						Location: "/data/test",
					}))

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
		),
	)
}
