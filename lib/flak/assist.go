package flak

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
)

// Methods that many executors might use

// Generates a temporary repeatr directory, creating all neccesary parent folders.
// Must be passed at least one directory name, all of which will be used in the path.
// Uses os.TempDir() to decide where to place.
//
// For example, GetTempDir("my-executor") -> /tmp/repeatr/my-executor/989443394
func GetTempDir(dirs ...string) string {

	if len(dirs) < 1 {
		panic(errors.ProgrammerError.New("Must have at least one sub-folder for tempdir"))
	}

	dir := []string{os.TempDir(), "repeatr"}
	dir = append(dir, dirs...)
	tempPath := filepath.Join(dir...)

	// Tempdir wants parent path to exist
	err := os.MkdirAll(tempPath, 0600)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	// Make temp dir for this instance
	folder, err := ioutil.TempDir(tempPath, "")
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	return folder
}

// Runs a function with a tempdir, cleaning up afterward.
func WithDir(f func(string), dirs ...string) {

	if len(dirs) < 1 {
		panic(errors.ProgrammerError.New("Must have at least one sub-folder for tempdir"))
	}

	tempPath := filepath.Join(dirs...)

	// Tempdir wants parent path to exist
	err := os.MkdirAll(tempPath, 0600)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	try.Do(func() {
		f(tempPath)
	}).Finally(func() {
		err := os.RemoveAll(tempPath)
		if err != nil {
			// TODO: we don't want to panic here, more like a debug log entry, "failed to remove tempdir."
			// Can accomplish once we add logging.
			panic(errors.IOError.Wrap(err))
		}
	}).Done()
}

func WaitAndHandleExit(cmd *exec.Cmd) int {
	exitCode := -1
	var err error
	for err == nil && exitCode == -1 {
		exitCode, err = WaitTry(cmd)
	}

	// Do one last Wait for good ol' times sake.  And to use the Cmd.closeDescriptors feature.
	cmd.Wait()

	return exitCode
}

// copious code copyforked from github.com/polydawn/pogo/gosh ... maybe we should just use it
func WaitTry(cmd *exec.Cmd) (int, error) {
	// The docs for os.Process.Wait() state "Wait waits for the Process to exit".
	// IT LIES.
	//
	// On unixy systems, under some states, os.Process.Wait() *also* returns for signals and other state changes.  See comments below, where waitStatus is being checked.
	// To actually wait for the process to exit, you have to Wait() repeatedly and check if the system-dependent codes are representative of real exit.
	//
	// You can *not* use os/exec.Cmd.Wait() to reliably wait for a command to exit on unix.  Can.  Not.  Do it.
	// os/exec.Cmd.Wait() explicitly sets a flag to see if you've called it before, and tells you to go to hell if you have.
	// Since Cmd.Wait() uses Process.Wait(), the latter of which cannot function correctly without repeated calls, and the former of which forbids repeated calls...
	// Yep, it's literally impossible to use os/exec.Cmd.Wait() correctly on unix.
	//
	processState, err := cmd.Process.Wait()
	if err != nil {
		return -1, err
	}

	if waitStatus, ok := processState.Sys().(syscall.WaitStatus); ok {
		if waitStatus.Exited() {
			return waitStatus.ExitStatus(), nil
		} else if waitStatus.Signaled() {
			// In bash, when a processs ends from a signal, the $? variable is set to 128+SIG.
			// We follow that same convention here.
			// So, a process terminated by ctrl-C returns 130.  A script that died to kill-9 returns 137.
			return int(waitStatus.Signal()) + 128, nil
		} else {
			// This should be more or less unreachable.
			//  ... the operative word there being "should".  Read: "you wish".
			// WaitStatus also defines Continued and Stopped states, but in practice, they don't (typically) appear here,
			//  because deep down, syscall.Wait4 is being called with options=0, and getting those states would require
			//  syscall.Wait4 being called with WUNTRACED or WCONTINUED.
			// However, syscall.Wait4 may also return the Continued and Stoppe states if ptrace() has been attached to the child,
			//  so, really, anything is possible here.
			// And thus, we have to return a special code here that causes wait to be tried in a loop.
			return -1, nil
		}
	} else {
		panic(errors.NotImplementedError.New("repeatr only works systems with posix-style process semantics."))
	}
}
