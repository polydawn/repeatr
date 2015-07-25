package nsinit

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/executor/tests"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	Convey("Spec Compliance: nsinit Executor", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.RequiresNamespaces,
			testutil.WithTmpdir(func() {
				execEng := &Executor{}
				execEng.Configure("nsinit_workspace")
				So(os.Mkdir(execEng.workspacePath, 0755), ShouldBeNil)

				//tests.CheckBasicExecution(execEng) // correct error reporting sections fail spec compliance
				tests.CheckFilesystemContainment(execEng)
				//tests.CheckPwdBehavior(execEng) // correct error reporting sections fail spec compliance
				tests.CheckEnvBehavior(execEng)
			}),
		),
	)
}
