package chroot

import (
	. "fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/basicjob"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/output"
)

var _ executor.Executor = &Executor{} // interface assertion

type Executor struct {
	workspacePath string
}

func (e *Executor) Configure(workspacePath string) {
	e.workspacePath = workspacePath
}

func (e *Executor) Start(f def.Formula) def.Job {

	// Prepare the forumla for execution on this host
	def.ValidateAll(&f)
	job := basicjob.New()

	go func() {
		// Run the formula in a temporary directory
		flak.WithDir(func(dir string) {
			job.Result = e.Run(f, job, dir)
		}, e.workspacePath, "job", string(job.Id()))

		// Directory is clean; job complete
		close(job.WaitChan)
	}()

	return job
}

// Executes a job, catching any panics.
func (e *Executor) Run(f def.Formula, j def.Job, d string) def.JobResult {
	var r def.JobResult

	try.Do(func() {
		r = e.Execute(f, j, d)
	}).Catch(executor.Error, func(err *errors.Error) {
		r.Error = err
	}).Catch(input.Error, func(err *errors.Error) {
		r.Error = err
	}).Catch(output.Error, func(err *errors.Error) {
		r.Error = err
	}).CatchAll(func(err error) {
		r.Error = executor.UnknownError.Wrap(err).(*errors.Error)
	}).Done()

	return r
}

// Execute a formula in a specified directory. MAY PANIC.
func (e *Executor) Execute(f def.Formula, j def.Job, d string) def.JobResult {

	result := def.JobResult{
		ID:       j.Id(),
		Error:    nil,
		ExitCode: 0, //TODO: gosh
		Outputs:  []def.Output{},
	}

	// Prepare filesystem
	rootfs := filepath.Join(d, "rootfs")
	flak.ProvisionInputs(f.Inputs, rootfs)
	flak.ProvisionOutputs(f.Outputs, rootfs)

	// chroot's are pretty easy.
	cmdName := f.Accents.Entrypoint[0]
	cmd := exec.Command(cmdName, f.Accents.Entrypoint[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:    rootfs,
		Pdeathsig: syscall.SIGKILL,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Println("Running formula...")
	if err := cmd.Start(); err != nil {
		if err2, ok := err.(*exec.Error); ok && err2.Err == exec.ErrNotFound {
			panic(executor.NoSuchCommandError.Wrap(err))
		}
		panic(executor.TaskExecError.Wrap(err))
	}

	// Wait for the job to complete
	result.ExitCode = flak.WaitAndHandleExit(cmd)

	// Save outputs
	result.Outputs = flak.PreserveOutputs(f.Outputs, rootfs)

	return result
}
