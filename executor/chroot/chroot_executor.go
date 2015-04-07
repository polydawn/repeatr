package chroot

import (
	"io"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/basicjob"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/lib/streamer"
	"polydawn.net/repeatr/output"
)

var _ executor.Executor = &Executor{} // interface assertion

type Executor struct {
	workspacePath string
}

func (e *Executor) Configure(workspacePath string) {
	e.workspacePath = workspacePath
}

func (e *Executor) Start(f def.Formula, id def.JobID) def.Job {

	// Prepare the forumla for execution on this host
	def.ValidateAll(&f)
	job := basicjob.New(id)

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
		ID:      j.Id(),
		Outputs: []def.Output{},
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

	// spool our output to a muxed stream
	var strm streamer.Mux
	strm = streamer.CborFileMux(filepath.Join(d, "log"))
	cmd.Stdin = nil
	cmd.Stdout = strm.Appender(1)
	cmd.Stderr = strm.Appender(2)
	j.(*basicjob.BasicJob).Reader = strm.Reader(1, 2)
	defer func() {
		// Close output streams.
		// (I thought exec should do this already...?  But doesn't seem to.)
		cmd.Stdout.(io.WriteCloser).Close()
		cmd.Stderr.(io.WriteCloser).Close()
	}()

	// launch execution.
	// transform gosh's typed errors to repeatr's hierarchical errors.
	var proc gosh.Proc
	try.Do(func() {
		proc = gosh.ExecProcCmd(cmd)
	}).CatchAll(func(err error) {
		switch err.(type) {
		case gosh.NoSuchCommandError:
			panic(executor.NoSuchCommandError.Wrap(err))
		case gosh.NoArgumentsErr:
			panic(executor.NoSuchCommandError.Wrap(err))
		case gosh.ProcMonitorError:
			panic(executor.TaskExecError.Wrap(err))
		default:
			panic(executor.UnknownError.Wrap(err))
		}
	}).Done()

	// Wait for the job to complete
	// REVIEW: consider exposing `gosh.Proc`'s interface as part of repeatr's job tracking api?
	result.ExitCode = proc.GetExitCode()

	// Save outputs
	result.Outputs = flak.PreserveOutputs(f.Outputs, rootfs)

	return result
}
