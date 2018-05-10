package examples

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	. "github.com/warpfork/go-wish"
)

func runTestcase(t *testing.T, tc testcase) {
	t.Helper()
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cmd := exec.CommandContext(ctx, tc.command()[0], tc.command()[1:]...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	processState, err := cmd.Process.Wait()
	if err != nil {
		panic(err)
	}
	if waitStatus, ok := processState.Sys().(syscall.WaitStatus); ok {
		if waitStatus.Exited() {
			Wish(t, waitStatus.ExitStatus(), ShouldEqual, tc.exitcode())
		} else if waitStatus.Signaled() {
			t.Errorf("process exited from signal %#v", waitStatus.Signal())
		} else {
			t.Errorf("process halted in terribly strange way")
		}
	}
	Wish(t, stdoutBuf.String(), ShouldEqual, strings.Join(tc.stdout(), "\n"))
	Wish(t, stderrBuf.String(), ShouldEqual, strings.Join(tc.stderr(), "\n"))
}

func TestAll(t *testing.T) {
	t.Skip("wip :)")
	tc := loadTestcase("hello.tcase")
	runTestcase(t, tc)
}
