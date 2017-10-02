package chroot

import (
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
	var (
		unpackTool rio.UnpackFunc = rioexecclient.UnpackFunc
		packTool   rio.PackFunc   = rioexecclient.PackFunc
	)

	WithTmpdir(func(tmpDir fs.AbsolutePath) {
		// Setup assembler and executor.  Both are reusable.
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
	})
}
