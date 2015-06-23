package chroot

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/executor/tests"
	"polydawn.net/repeatr/testutil"
)

func Test(t *testing.T) {
	Convey("Spec Compliance: Chroot Executor", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.WithTmpdir(func() {
				execEng := &Executor{
					workspacePath: "chroot_workspace",
				}
				So(os.Mkdir(execEng.workspacePath, 0755), ShouldBeNil)

				tests.CheckBasicExecution(execEng)
				tests.CheckFilesystemContainment(execEng)
				tests.CheckPwdBehavior(execEng)
				tests.CheckEnvBehavior(execEng)
			}),
		),
	)
}
