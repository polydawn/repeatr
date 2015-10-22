package runc

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/executor/tests"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	Convey("Spec Compliance: Runc Executor", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.RequiresNamespaces,
			testutil.WithTmpdir(func() {
				execEng := &Executor{}
				execEng.Configure("runc_workspace")
				So(os.Mkdir(execEng.workspacePath, 0755), ShouldBeNil)

				tests.CheckBasicExecution(execEng)
				tests.CheckFilesystemContainment(execEng)
				tests.CheckPwdBehavior(execEng)
				tests.CheckEnvBehavior(execEng)
			}),
		),
	)
}
