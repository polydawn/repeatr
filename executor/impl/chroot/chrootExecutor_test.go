package chroot

import (
	"os"
	"testing"

	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/repeatr/executor/tests"
	. "go.polydawn.net/repeatr/testutil"
	"go.polydawn.net/rio/client"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fs/osfs"
	"go.polydawn.net/rio/stitch"
)

func TestChrootExecutor(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("the chroot executor requires root privs")
	}

	var (
		unpackTool rio.UnpackFunc = rioexecclient.UnpackFunc
		packTool   rio.PackFunc   = rioexecclient.PackFunc
	)

	WithTmpdir(func(tmpDir fs.AbsolutePath) {
		// Setup assembler and executor.  Both are reusable.
		//  Use env to communicate our test tempdir down to Rio.
		os.Setenv("RIO_BASE", tmpDir.String()+"/rio")
		asm, err := stitch.NewAssembler(unpackTool)
		AssertNoError(t, err)
		exe := Executor{
			osfs.New(tmpDir.Join(fs.MustRelPath("ws"))),
			asm,
			packTool,
		}

		tests.CheckHelloWorldTxt(t, exe.Run)
		tests.CheckRoundtripRootfs(t, exe.Run)
		tests.CheckReportingExitCodes(t, exe.Run)
		tests.CheckSettingCwd(t, exe.Run)
		tests.CheckErrorFromUnfetchableWares(t, exe.Run)
		tests.CheckUserinfoDefault(t, exe.Run)
		tests.CheckAdvancedUserinfo(t, exe.Run)
		tests.CheckRootyUserinfo(t, exe.Run)
	})
}
