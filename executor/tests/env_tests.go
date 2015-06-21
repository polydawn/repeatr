package tests

import (
	"io/ioutil"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/guid"
)

func CheckPwdBehavior(execEng executor.Executor) {
	Convey("SPEC: Working directory should be contained", func() {
		formula := getBaseFormula()

		Convey("The default pwd should be the root", func() {
			// This test exists because of a peculularly terrifying fact about chroots:
			// If you spawn a new process with the chroot without setting its current working dir,
			// the cwd is still *whatever it inherits*.  And even if the process can't reach there
			// starting from "/", it can still *walk deeper from that cwd*.
			formula.Accents = def.Accents{
				//Entrypoint: []string{"find", "-maxdepth", "5"}, // if you goofed, during test runs this will show you the executor's workspace!
				//Entrypoint: []string{"bash", "-c", "find -maxdepth 5 ; echo --- ; echo \"$PWD\" ; cd \"$(echo \"$PWD\" | sed 's/^(unreachable)//')\" ; ls"}, // this demo's that you can't actually cd back to it, though.
				Entrypoint: []string{"pwd"},
			}

			job := execEng.Start(formula, def.JobID(guid.New()), ioutil.Discard)
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldBeNil)
			So(job.Wait().ExitCode, ShouldEqual, 0)
			msg, err := ioutil.ReadAll(job.OutputReader())
			So(err, ShouldBeNil)
			So(string(msg), ShouldEqual, "/\n")
		})
	})
}
