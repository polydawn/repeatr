package tests

import (
	"io/ioutil"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/testutil/filefixture"
)

func CheckFilesystemContainment(execEng executor.Executor) {
	Convey("SPEC: other inputs can be seen, in place", func() {
		formula := getBaseFormula()

		Convey("Launch should succeed", func() {
			filefixture.Beta.Create("./fixture/beta")
			formula.Inputs = append(formula.Inputs, (def.Input{
				Type:     "dir",
				Hash:     filefixture.Beta_Hash,
				URI:      "./fixture/beta",
				Location: "/data/test",
			}))

			formula.Accents = def.Accents{
				Entrypoint: []string{"/bin/true"},
			}
			job := execEng.Start(formula, def.JobID(guid.New()), ioutil.Discard)
			So(job, ShouldNotBeNil)
			So(job.Wait().Error, ShouldBeNil)
			So(job.Wait().ExitCode, ShouldEqual, 0)

			Convey("Commands inside the job should be able to see the mounted files", FailureContinues, func() {
				formula.Accents = def.Accents{
					Entrypoint: []string{"ls", "/data/test"},
				}

				job := execEng.Start(formula, def.JobID(guid.New()), ioutil.Discard)
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
