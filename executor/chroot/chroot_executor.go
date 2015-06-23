package chroot

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/basicjob"
	"polydawn.net/repeatr/executor/util"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/lib/streamer"
)

var _ executor.Executor = &Executor{} // interface assertion

type Executor struct {
	workspacePath string
}

func (e *Executor) Configure(workspacePath string) {
	e.workspacePath = workspacePath
}

func (e *Executor) Start(f def.Formula, id def.JobID, journal io.Writer) def.Job {

	// Prepare the forumla for execution on this host
	def.ValidateAll(&f)

	job := basicjob.New(id)
	jobReady := make(chan struct{})

	go func() {
		// Run the formula in a temporary directory
		flak.WithDir(func(dir string) {

			// spool our output to a muxed stream
			var strm streamer.Mux
			strm = streamer.CborFileMux(filepath.Join(dir, "log"))
			outS := strm.Appender(1)
			errS := strm.Appender(2)
			job.Reader = strm.Reader(1, 2)
			defer func() {
				// Regardless of how the job ends (or even if it fails the remaining setup), output streams must be terminated.
				outS.Close()
				errS.Close()
			}()

			// Job is ready to stream process output
			close(jobReady)

			job.Result = e.Run(f, job, dir, outS, errS, journal)
		}, e.workspacePath, "job", string(job.Id()))

		// Directory is clean; job complete
		close(job.WaitChan)
	}()

	<-jobReady
	return job
}

// Executes a job, catching any panics.
func (e *Executor) Run(f def.Formula, j def.Job, d string, outS, errS io.WriteCloser, journal io.Writer) def.JobResult {
	r := def.JobResult{
		ID:       j.Id(),
		ExitCode: -1,
	}

	try.Do(func() {
		e.Execute(f, j, d, &r, outS, errS, journal)
	}).Catch(executor.Error, func(err *errors.Error) {
		r.Error = err
	}).Catch(integrity.Error, func(err *errors.Error) {
		r.Error = err
	}).CatchAll(func(err error) {
		r.Error = executor.UnknownError.Wrap(err).(*errors.Error)
	}).Done()

	return r
}

// Execute a formula in a specified directory. MAY PANIC.
func (e *Executor) Execute(f def.Formula, j def.Job, d string, result *def.JobResult, outS, errS io.WriteCloser, journal io.Writer) {
	// Prepare filesystem
	rootfs := filepath.Join(d, "rootfs")
	transmat := util.DefaultTransmat()
	assembly := util.ProvisionInputs(
		transmat,
		util.BestAssembler(),
		f.Inputs, rootfs, journal,
	)
	defer assembly.Teardown() // What ever happens: Disassemble filesystem
	util.ProvisionOutputs(f.Outputs, rootfs, journal)

	// chroot's are pretty easy.
	cmdName := f.Accents.Entrypoint[0]
	cmd := exec.Command(cmdName, f.Accents.Entrypoint[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:    rootfs,
		Pdeathsig: syscall.SIGKILL,
	}

	// except handling cwd is a little odd.
	// see comments in gosh tests with chroot for information about the odd behavior we're hacking around here;
	// we're comfortable making this special check here, but not upstreaming it to gosh, because in our context we "know" we're not racing anyone.
	if externalCwdStat, err := os.Stat(filepath.Join(rootfs, f.Accents.Cwd)); err != nil {
		panic(executor.TaskExecError.New("cannot set cwd to %q: %s", f.Accents.Cwd, err.(*os.PathError).Err))
	} else if !externalCwdStat.IsDir() {
		panic(executor.TaskExecError.New("cannot set cwd to %q: not a dir", f.Accents.Cwd))
	}
	cmd.Dir = f.Accents.Cwd

	// set env.
	// initialization already required by earlier 'validate' calls.
	cmd.Env = envToSlice(f.Accents.Env)

	cmd.Stdin = nil
	cmd.Stdout = outS
	cmd.Stderr = errS

	// launch execution.
	// transform gosh's typed errors to repeatr's hierarchical errors.
	var proc gosh.Proc
	try.Do(func() {
		proc = gosh.ExecProcCmd(cmd)
	}).CatchAll(func(err error) {
		switch err.(type) {
		case gosh.NoSuchCommandError:
			panic(executor.NoSuchCommandError.Wrap(err))
		case gosh.NoArgumentsError:
			panic(executor.NoSuchCommandError.Wrap(err))
		case gosh.NoSuchCwdError: // included for clarity and completeness, but we'll never actually see this; see comments in gosh about the interaction of chroot and cwd error handling.
			panic(executor.TaskExecError.Wrap(err))
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
	result.Outputs = util.PreserveOutputs(transmat, f.Outputs, rootfs, journal)
}

func envToSlice(env map[string]string) []string {
	rv := make([]string, len(env))
	i := 0
	for k, v := range env {
		rv[i] = k + "=" + v
		i++
	}
	return rv
}
