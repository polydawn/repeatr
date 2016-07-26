package tests

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/inconshreveable/log15"
	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/lib/testutil"
)

func CheckPwdBehavior(execEng executor.Executor) {
	Convey("SPEC: Working directory should be contained", func(c C) {
		formula := getBaseFormula()

		Convey("The default pwd should be /task", func() {
			// This test exists because of a peculularly terrifying fact about chroots:
			// If you spawn a new process with the chroot without setting its current working dir,
			// the cwd is still *whatever it inherits*.  And even if the process can't reach there
			// starting from "/", it can still *walk deeper from that cwd*.
			// (This is now more of a historical scare detail since we've changed the default
			// behavior to be a specific directory, but still interesting.)
			formula.Action = def.Action{
				//Entrypoint: []string{"find", "-maxdepth", "5"}, // if you goofed, during test runs this will show you the executor's workspace!
				//Entrypoint: []string{"bash", "-c", "find -maxdepth 5 ; echo --- ; echo \"$PWD\" ; cd \"$(echo \"$PWD\" | sed 's/^(unreachable)//')\" ; ls"}, // this demo's that you can't actually cd back to it, though.
				Entrypoint: []string{"pwd"},
			}

			soExpectSuccessAndOutput(execEng, formula, testutil.TestLogger(c),
				"/task\n",
			)
		})

		Convey("Setting another cwd should work", func() {
			formula.Action = def.Action{
				Cwd:        "/usr",
				Entrypoint: []string{"pwd"},
			}

			soExpectSuccessAndOutput(execEng, formula, testutil.TestLogger(c),
				"/usr\n",
			)
		})

		Convey("Setting a nonexistent cwd...", func() {
			formula.Action = def.Action{
				Cwd:        "/does/not/exist/by/any/means",
				Entrypoint: []string{"pwd"},
			}

			Convey("should succeed (since by default cradle creates it)", FailureContinues, func() {
				soExpectSuccessAndOutput(execEng, formula, testutil.TestLogger(c),
					"/does/not/exist/by/any/means\n",
				)
			})

			Convey("should fail to launch if cradle is disabled", FailureContinues, func() {
				var _false bool
				formula.Action.Cradle = &_false

				job := execEng.Start(formula, executor.JobID(guid.New()), nil, testutil.TestLogger(c))
				So(job, ShouldNotBeNil)
				So(job.Wait().Error, ShouldNotBeNil)
				So(job.Wait().Error, testutil.ShouldBeErrorClass, executor.NoSuchCwdError)
				So(job.Wait().ExitCode, ShouldEqual, -1)
				msg, err := ioutil.ReadAll(job.OutputReader())
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "")
			})
		})

		Convey("Setting a non-dir cwd should fail to launch", FailureContinues, func() {
			formula.Action = def.Action{
				Cwd:        "/bin/sh",
				Entrypoint: []string{"pwd"},
			}

			job := execEng.Start(formula, executor.JobID(guid.New()), nil, testutil.TestLogger(c))
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldNotBeNil)
			So(job.Wait().Error, testutil.ShouldBeErrorClass, executor.NoSuchCwdError)
			So(job.Wait().ExitCode, ShouldEqual, -1)
			msg, err := ioutil.ReadAll(job.OutputReader())
			So(err, ShouldBeNil)
			So(string(msg), ShouldEqual, "")
		})
	})
}

func CheckEnvBehavior(execEng executor.Executor) {
	Convey("SPEC: Env vars should be contained", func(c C) {
		formula := getBaseFormula()
		formula.Action = def.Action{
			Entrypoint: []string{"env"},
		}

		Convey("Env from the parent should not be inherited", func() {
			os.Setenv("REPEATR_TEST_KEY_1", "test value")
			defer os.Unsetenv("REPEATR_TEST_KEY_1") // using unique strings per test anyway, because this is too scary

			job := execEng.Start(formula, executor.JobID(guid.New()), nil, testutil.TestLogger(c))
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldBeNil)
			So(job.Wait().ExitCode, ShouldEqual, 0)
			msg, err := ioutil.ReadAll(job.OutputReader())
			So(err, ShouldBeNil)
			So(strings.Contains("REPEATR_TEST_KEY_1", string(msg)), ShouldBeFalse)
		})

		Convey("Env specified with the job should be applied", func() {
			formula.Action.Env = make(map[string]string)
			formula.Action.Env["REPEATR_TEST_KEY_2"] = "test value"

			job := execEng.Start(formula, executor.JobID(guid.New()), nil, testutil.TestLogger(c))
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldBeNil)
			So(job.Wait().ExitCode, ShouldEqual, 0)
			msg, err := ioutil.ReadAll(job.OutputReader())
			So(err, ShouldBeNil)
			So(string(msg), ShouldContainSubstring, "REPEATR_TEST_KEY_2=test value")
		})

	})
}

func CheckHostnameBehavior(execEng executor.Executor) {
	Convey("SPEC: Hostname should be job ID by default", func(c C) {
		// note: considered just saying "not the host", but figured we
		//  might as well pick a stance and stick with it.
		formula := getBaseFormula()
		formula.Action = def.Action{
			Entrypoint: []string{"hostname"},
		}

		jobID := executor.JobID(guid.New())
		job := execEng.Start(formula, jobID, nil, testutil.TestLogger(c))
		So(job, ShouldNotBeNil)
		So(job.Wait().Error, ShouldBeNil)
		So(job.Wait().ExitCode, ShouldEqual, 0)
		msg, err := ioutil.ReadAll(job.OutputReader())
		So(err, ShouldBeNil)
		So(string(msg), ShouldEqual, string(jobID)+"\n")
	})

	Convey("SPEC: Hostname obey formula if set", func(c C) {
		formula := getBaseFormula()
		formula.Action = def.Action{
			Entrypoint: []string{"hostname"},
		}
		formula.Action.Hostname = "a-custom-hostname"

		jobID := executor.JobID(guid.New())
		job := execEng.Start(formula, jobID, nil, testutil.TestLogger(c))
		So(job, ShouldNotBeNil)
		So(job.Wait().Error, ShouldBeNil)
		So(job.Wait().ExitCode, ShouldEqual, 0)
		msg, err := ioutil.ReadAll(job.OutputReader())
		So(err, ShouldBeNil)
		So(string(msg), ShouldEqual, formula.Action.Hostname+"\n")
	})
}

func soExpectSuccessAndOutput(execEng executor.Executor, formula def.Formula, log log15.Logger, output string) {
	job := execEng.Start(formula, executor.JobID(guid.New()), nil, log)
	So(job, ShouldNotBeNil)
	So(job.Wait().Error, ShouldBeNil)
	So(job.Wait().ExitCode, ShouldEqual, 0)
	msg, err := ioutil.ReadAll(job.OutputReader())
	So(err, ShouldBeNil)
	So(string(msg), ShouldEqual, output)
}
