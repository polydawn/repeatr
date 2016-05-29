package tests

import (
	"io/ioutil"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/lib/testutil"
	"polydawn.net/repeatr/lib/testutil/filefixture"
)

func CheckFilesystemContainment(execEng executor.Executor) {
	Convey("SPEC: Launching with multiple inputs should work", func(c C) {
		formula := getBaseFormula()

		Convey("Launch should succeed", func() {
			filefixture.Beta.Create("./fixture/beta")
			formula.Inputs["part2"] = &def.Input{
				Type:       "dir",
				Hash:       filefixture.Beta_Hash,
				Warehouses: []string{"file://./fixture/beta"},
				MountPath:  "/data/test",
			}

			formula.Action = def.Action{
				Entrypoint: []string{"/bin/true"},
			}
			job := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c})
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldBeNil)
			So(job.Wait().ExitCode, ShouldEqual, 0)

			Convey("Commands inside the job should be able to see the mounted files", FailureContinues, func() {
				formula.Action = def.Action{
					Entrypoint: []string{"ls", "/data/test"},
				}

				job := execEng.Start(formula, def.JobID(guid.New()), nil, testutil.Writer{c})
				So(job, ShouldNotBeNil)
				So(job.Wait().Error, ShouldBeNil)
				So(job.Wait().ExitCode, ShouldEqual, 0)
				msg, err := ioutil.ReadAll(job.OutputReader())
				So(err, ShouldBeNil)
				So(string(msg), ShouldEqual, "1\n2\n3\n")
			})
		})
	})
}
