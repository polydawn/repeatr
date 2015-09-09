package tests

import (
	"io/ioutil"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/testutil"
)

/*
	Check the basics of exec:
	  - Does it work at all?
	  - Can we see error codes?
	  - Can we stream stdout?
	  - If there's no rootfs, does it panic?
	  - If the command's not found, does it panic?

	Anything that requires *more than the one rootfs input* or anything
	about outputs belongs in another part of the spec tests.

	If any of these fail, most other parts of the specs will also fail.
*/
func CheckBasicExecution(execEng executor.Executor) {
	Convey("SPEC: Attempting launch with a rootfs that doesn't exist should error", func() {
		formula := def.Formula{
			Inputs: []def.Input{
				{
					Type:     "tar",
					Location: "/",
					// Funny thing is, the URI isn't even necessarily where the buck stops;
					// Remote URIs need not be checked if caches are in play, etc.
					// So the hash needs to be set (and needs to be invalid).
					URI:  "file:///nonexistance/in/its/most/essential/unform.tar.gz",
					Hash: "defnot",
				},
			},
		}

		Convey("We should get an error from the warehouse", func() {
			result := execEng.Start(formula, def.JobID(guid.New()), nil, ioutil.Discard).Wait()
			So(result.Error, testutil.ShouldBeErrorClass, integrity.WarehouseError)
		})

		Convey("The job exit code should clearly indicate failure", FailureContinues, func() {
			formula.Action = def.Action{
				Entrypoint: []string{"echo", "echococo"},
			}
			job := execEng.Start(formula, def.JobID(guid.New()), nil, ioutil.Discard)
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldNotBeNil)
			// Even though one should clearly also check the error status,
			//  zero here could be very confusing, so jobs that error before start should be -1.
			So(job.Wait().ExitCode, ShouldEqual, -1)
		})
	})

	Convey("SPEC: Launching a command with a working rootfs should work", func() {
		formula := getBaseFormula()

		Convey("The executor should be able to invoke echo", FailureContinues, func() {
			formula.Action = def.Action{
				Entrypoint: []string{"echo", "echococo"},
			}

			job := execEng.Start(formula, def.JobID(guid.New()), nil, ioutil.Discard)
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
			formula.Action = def.Action{
				Entrypoint: []string{"sh", "-c", "exit 14"},
			}

			job := execEng.Start(formula, def.JobID(guid.New()), nil, ioutil.Discard)
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldBeNil)
			So(job.Wait().ExitCode, ShouldEqual, 14)
		})

		Convey("The executor should report command not found clearly", FailureContinues, func() {
			formula.Action = def.Action{
				Entrypoint: []string{"not a command"},
			}

			job := execEng.Start(formula, def.JobID(guid.New()), nil, ioutil.Discard)
			So(job.Wait().Error, testutil.ShouldBeErrorClass, executor.NoSuchCommandError)
			So(job.Wait().ExitCode, ShouldEqual, -1)
			msg, err := ioutil.ReadAll(job.OutputReader())
			So(err, ShouldBeNil)
			So(string(msg), ShouldEqual, "")
		})
	})
}
