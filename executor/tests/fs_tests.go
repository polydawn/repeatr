package tests

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/testutil/filefixture"
)

func CheckFilesystemContainment(execEng executor.Executor) {
	// find local assets.  we rely on local files bootstrapped by earlier build process steps rather than have executor tests depend on networked transmats (and thus *network*).
	// is janky.  don't know of a best practice for finding your "project dir".
	projPath, _ := os.Getwd()
	projPath = filepath.Dir(filepath.Dir(projPath))

	Convey("SPEC: other inputs can be seen, in place", func() {
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
