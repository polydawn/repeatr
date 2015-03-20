package chroot

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/input/dispatch"
	"polydawn.net/repeatr/lib/guid"
)

type Executor struct {
	workspacePath string // default: something like '/var/lib/repeatr/executors/chroot/'.
}

func (x *Executor) Run(formula def.Formula) (job def.Job, outs []def.Output) {
	try.Do(func() {
		job, outs = x.run(formula)
	}).Catch(input.InputError, func(e *errors.Error) {
		// REVIEW: also directly pass input/output system errors up?  or, since we may have to gather several, put them in a group and wrap them in a "prereqs failed" executor error?
		panic(e)
	}).Catch(Error, func(e *errors.Error) {
		panic(e)
	}).CatchAll(func(err error) {
		panic(UnknownError.Wrap(err))
	}).Done()
	return
}

func (x *Executor) run(formula def.Formula) (def.Job, []def.Output) {
	// Prepare the forumla for execution on this host
	def.ValidateAll(&formula)

	// make up a job id
	jobID := def.JobID(guid.New())

	// make a rootfs in our workspace using the jobID
	rootfsPath := filepath.Join(x.workspacePath, string(jobID))
	if err := os.Mkdir(rootfsPath, 0755); err != nil {
		panic(Error.Wrap(errors.IOError.Wrap(err))) // REVIEW: WorkspaceIOError?  or a flag that indicates "wow, super hosed"?
	}

	// prep inputs
	x.prepareInputs(rootfsPath, formula.Inputs)

	// prep outputs
	// TODO implement some outputs!

	// sandbox up and invoke the real job
	job := x.invokeTask(rootfsPath, formula)

	// commit outputs
	// TODO implement some outputs!

	return job, nil
}

func (x *Executor) prepareInputs(rootfsPath string, inputs []def.Input) {
	for _, input := range inputs {
		path := filepath.Join(rootfsPath, input.Location)

		// Ensure that the parent folder of this input exists
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			panic(Error.Wrap(errors.IOError.Wrap(err)))
		}

		// Run input
		// TODO: all of them, asynchronously.
		err = <-inputdispatch.Get(input).Apply(path)
		if err != nil {
			panic(err)
		}
	}
}

func (x *Executor) invokeTask(rootfsPath string, formula def.Formula) def.Job {
	// REVIEW: method sig kind of hints we should gather all the "task" related parts together so we don't have to pass the whole formula here

	// chroot's are pretty easy.
	cmdName := formula.Accents.Entrypoint[0]
	cmd := exec.Command(cmdName, formula.Accents.Entrypoint[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:    rootfsPath,
		Pdeathsig: syscall.SIGKILL,
	}

	// set up collection and job reporting
	wait := make(chan struct{})
	job := &job{wait: wait}
	cmd.Stdin = nil
	cmd.Stdout = &job.buf
	cmd.Stderr = &job.buf

	if err := cmd.Start(); err != nil {
		// TODO: i'd love to report executable-not-found (very) differently from other major blowups, syscall fails, etc.
		panic(TaskExecError.Wrap(err))
	}

	// spawn a routine to collect and wait
	go func() {
		job.waitAndHandleExit(cmd)
		close(wait)
	}()

	return job
}

type job struct {
	wait     <-chan struct{}
	buf      bytes.Buffer
	exitCode int
}

func (j job) OutputReader() io.Reader {
	return &j.buf
}

func (j *job) ExitCode() int {
	<-j.wait
	return j.exitCode
}

func (j *job) waitAndHandleExit(cmd *exec.Cmd) {
	exitCode := -1
	var err error
	for err == nil && exitCode == -1 {
		exitCode, err = waitTry(cmd)
	}

	// Do one last Wait for good ol' times sake.  And to use the Cmd.closeDescriptors feature.
	cmd.Wait()

	j.exitCode = exitCode
}

// copious code copyforked from github.com/polydawn/pogo/gosh ... maybe we should just use it
func waitTry(cmd *exec.Cmd) (int, error) {
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
