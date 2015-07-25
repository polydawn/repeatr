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
			testutil.WithTmpdir(func() {
				execEng := &Executor{}
				execEng.Configure("nsinit_workspace")
				So(os.Mkdir(execEng.workspacePath, 0755), ShouldBeNil)

				tests.CheckBasicExecution(execEng)
				tests.CheckFilesystemContainment(execEng)
				tests.CheckPwdBehavior(execEng)
				tests.CheckEnvBehavior(execEng)
			}),
		),
	)
}
