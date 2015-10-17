package tests

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/testutil"
)

func CheckPwdBehavior(execEng executor.Executor) {
	Convey("SPEC: Working directory should be contained", func(c C) {
		formula := getBaseFormula()

		Convey("The default pwd should be the root", func() {
			// This test exists because of a peculularly terrifying fact about chroots:
			// If you spawn a new process with the chroot without setting its current working dir,
			// the cwd is still *whatever it inherits*.  And even if the process can't reach there
			// starting from "/", it can still *walk deeper from that cwd*.
			formula.Action = def.Action{
				//Entrypoint: []string{"find", "-maxdepth", "5"}, // if you goofed, during test runs this will show you the executor's workspace!
				//Entrypoint: []string{"bash", "-c", "find -maxdepth 5 ; echo --- ; echo \"$PWD\" ; cd \"$(echo \"$PWD\" | sed 's/^(unreachable)//')\" ; ls"}, // this demo's that you can't actually cd back to it, though.
				Entrypoint: []string{"pwd"},
			}

			soExpectSuccessAndOutput(execEng, formula, testutil.Writer{c},
				"/\n",
			)
		})

		Convey("Setting another cwd should work", func() {
			formula.Action = def.Action{
				Cwd:        "/usr",
				Entrypoint: []string{"pwd"},
			}

			soExpectSuccessAndOutput(execEng, formula, testutil.Writer{c},
				"/usr\n",
			)
		})

		Convey("Setting a nonexistent cwd should fail to launch", FailureContinues, func() {
			formula.Action = def.Action{
				Cwd:        "/does/not/exist/by/any/means",
				Entrypoint: []string{"pwd"},
			}

			job := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c})
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldNotBeNil)
			So(job.Wait().Error, testutil.ShouldBeErrorClass, executor.TaskExecError)
			So(job.Wait().ExitCode, ShouldEqual, -1)
			msg, err := ioutil.ReadAll(job.OutputReader())
			So(err, ShouldBeNil)
			So(string(msg), ShouldEqual, "")
		})
	})
}

func CheckEnvBehavior(execEng executor.Executor) {
	// NOTE the chroot executor currently uses `def.ValidateAll` which effectively *does not permit you to have an empty $PATH var*.
	// we may decide to change that later -- but there's a correctness versus convenience battle there, and for now, we're going with convenience.

	Convey("SPEC: Env vars should be contained", func(c C) {
		formula := getBaseFormula()
		formula.Action = def.Action{
			Entrypoint: []string{"env"},
		}

		Convey("Env from the parent should not be inherited", func() {
			os.Setenv("REPEATR_TEST_KEY_1", "test value")
			defer os.Unsetenv("REPEATR_TEST_KEY_1") // using unique strings per test anyway, because this is too scary

			job := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c})
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

			job := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c})
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldBeNil)
			So(job.Wait().ExitCode, ShouldEqual, 0)
			msg, err := ioutil.ReadAll(job.OutputReader())
			So(err, ShouldBeNil)
			So(string(msg), ShouldContainSubstring, "REPEATR_TEST_KEY_2=test value")
		})

	})
}

func soExpectSuccessAndOutput(execEng executor.Executor, formula def.Formula, journal io.Writer, output string) {
	job := execEng.Start(formula, def.JobID(guid.New()), nil, journal)
	So(job, ShouldNotBeNil)
	So(job.Wait().Error, ShouldBeNil)
	So(job.Wait().ExitCode, ShouldEqual, 0)
	msg, err := ioutil.ReadAll(job.OutputReader())
	So(err, ShouldBeNil)
	So(string(msg), ShouldEqual, output)
}
