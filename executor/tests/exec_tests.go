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

func CheckBasicExecution(execEng executor.Executor) {
	Convey("SPEC: a rootfs input that doesn't exist raises errors", func() {
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
			result := execEng.Start(formula, def.JobID(guid.New()), ioutil.Discard).Wait()
			So(result.Error, testutil.ShouldBeErrorClass, integrity.WarehouseError)
		})

		Convey("The job exit code should clearly indicate failure", FailureContinues, func() {
			formula.Accents = def.Accents{
				Entrypoint: []string{"echo", "echococo"},
			}
			job := execEng.Start(formula, def.JobID(guid.New()), ioutil.Discard)
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldNotBeNil)
			// Even though one should clearly also check the error status,
			//  zero here could be very confusing, so jobs that error before start should be -1.
			So(job.Wait().ExitCode, ShouldEqual, -1)
		})
	})
}
