package examples

import (
	"bytes"
	"context"
	"os"
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
	os.Setenv("PATH", os.Getenv("GOBIN"))
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
	if expected := tc.stdout(); expected != nil {
		actual := strings.Split(stdoutBuf.String(), "\n")
		actual = paveRunrecords(actual)
		if os.Getenv("REFRESH_FIXTURES") != "" {
			expected = actual
			tc.hunks.PutSection("stdout", []byte(strings.Join(expected, "\n")))
		}
		Wish(t, strings.Join(actual, "\n"), ShouldEqual, strings.Join(expected, "\n"))
	}
	if expected := tc.stderr(); expected != nil {
		actual := strings.Split(stderrBuf.String(), "\n")
		actual = paveAnsicolors(actual)
		actual = paveLogtimes(actual)
		if os.Getenv("REFRESH_FIXTURES") != "" {
			expected = actual
			tc.hunks.PutSection("stderr", []byte(strings.Join(expected, "\n")))
		}
		Wish(t, strings.Join(actual, "\n"), ShouldEqual, strings.Join(expected, "\n"))
	}
	if os.Getenv("REFRESH_FIXTURES") != "" {
		tc.saveHunks()
	}
}

func TestAll(t *testing.T) {
	tc := loadTestcase("hello.tcase")
	runTestcase(t, tc)
}
