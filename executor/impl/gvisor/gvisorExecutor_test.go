package gvisor

import (
	"os"
	"testing"

	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/repeatr/executor/tests"
	. "go.polydawn.net/repeatr/testutil"
	"go.polydawn.net/rio/client"
	"go.polydawn.net/rio/fs"
)

func TestGvisorExecutor(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("the runc executor requires root privs")
	}

	var (
		unpackTool rio.UnpackFunc = rioexecclient.UnpackFunc
		packTool   rio.PackFunc   = rioexecclient.PackFunc
	)

	WithTmpdir(func(tmpDir fs.AbsolutePath) {
		// Setup assembler and executor.  Both are reusable.
		//  Use env to communicate our test tempdir down to Rio.
		os.Setenv("RIO_BASE", tmpDir.String()+"/rio")
		runTool, err := NewExecutor(
			tmpDir.Join(fs.MustRelPath("ws")),
			unpackTool,
			packTool,
		)
		AssertNoError(t, err)

		tests.CheckHelloWorldTxt(t, runTool)
		tests.CheckRoundtripRootfs(t, runTool)
		tests.CheckReportingExitCodes(t, runTool)
		tests.CheckErrorFromUnfetchableWares(t, runTool)
		tests.CheckUserinfoDefault(t, runTool)
		tests.CheckAdvancedUserinfo(t, runTool)
		tests.CheckRootyUserinfo(t, runTool)
	})
}
