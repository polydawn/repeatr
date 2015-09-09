package nsinit

import (
	"io"
	"os/exec"
	"path/filepath"

	"github.com/inconshreveable/log15"
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

// interface assertion
var _ executor.Executor = &Executor{}

type Executor struct {
	workspacePath string
}

func (e *Executor) Configure(workspacePath string) {
	var err error
	// immediately convert path to absolute.  nsinit rejects non-abs paths.
	e.workspacePath, err = filepath.Abs(workspacePath)
	if err != nil {
		panic(executor.ConfigError.New("could not use workspace path %q: %s", workspacePath, err))
	}
}

func (e *Executor) Start(f def.Formula, id def.JobID, stdin io.Reader, journal io.Writer) def.Job {

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
			job.Streams = strm
			defer func() {
				// Regardless of how the job ends (or even if it fails the remaining setup), output streams must be terminated.
				outS.Close()
				errS.Close()
			}()

			// Job is ready to stream process output
			close(jobReady)

			// Set up a logger.  Tag all messages with this jobid.
			logger := log15.New(log15.Ctx{"JobID": id})
			logger.SetHandler(log15.StreamHandler(journal, log15.TerminalFormat()))

			job.Result = e.Run(f, job, dir, stdin, outS, errS, logger)
		}, e.workspacePath, "job", string(job.Id()))

		// Directory is clean; job complete
		close(job.WaitChan)
	}()

	<-jobReady
	return job
}

// Executes a job, catching any panics.
func (e *Executor) Run(f def.Formula, j def.Job, d string, stdin io.Reader, outS, errS io.WriteCloser, journal log15.Logger) def.JobResult {
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
func (e *Executor) Execute(f def.Formula, j def.Job, d string, result *def.JobResult, outS, errS io.WriteCloser, journal log15.Logger) {
	// Dedicated rootfs folder to distinguish container from nsinit noise
	rootfs := filepath.Join(d, "rootfs")

	// nsinit wants to have a logfile
	logFile := filepath.Join(d, "nsinit-debug.log")

	// Prep command
	args := []string{}

	// Global options:
	// --root will place the 'nsinit' folder (holding a state.json file) in d
	// --log-file does much the same with a log file (unsure if care?)
	// --debug allegedly enables debug output in the log file
	args = append(args, "--root", d, "--log-file", logFile, "--debug")

	// Subcommand, and tell nsinit to not desire a JSON file (instead just use many flergs)
	args = append(args, "exec", "--create")

	// Use the host's networking (no bridge, no namespaces, etc)
	args = append(args, "--net=host")

	// Where our system image exists
	args = append(args, "--rootfs", rootfs)

	// Set cwd
	args = append(args, "--cwd", f.Action.Cwd)

	// Add all desired environment variables
	for k, v := range f.Action.Env {
		args = append(args, "--env", k+"="+v)
	}

	// Unroll command args
	args = append(args, f.Action.Entrypoint...)

	// Prepare command to exec
	cmd := exec.Command("nsinit", args...)

	cmd.Stdin = nil
	cmd.Stdout = outS
	cmd.Stderr = errS

	// Prepare filesystem
	transmat := util.DefaultTransmat()
	assembly := util.ProvisionInputs(
		transmat,
		util.BestAssembler(),
		f.Inputs, rootfs, journal,
	)
	defer assembly.Teardown() // What ever happens: Disassemble filesystem
	util.ProvisionOutputs(f.Outputs, rootfs, journal)

	// launch execution.
	// transform gosh's typed errors to repeatr's hierarchical errors.
	// this is... not untroubled code: since we're invoking a helper that's then
	//  proxying the exec even further, most errors are fatal (the mapping here is
	//   very different than in e.g. chroot executor, and provides much less meaning).
	var proc gosh.Proc
	try.Do(func() {
		proc = gosh.ExecProcCmd(cmd)
	}).CatchAll(func(err error) {
		switch err.(type) {
		case gosh.NoSuchCommandError:
			panic(executor.ConfigError.New("nsinit binary is missing"))
		case gosh.NoArgumentsError:
			panic(executor.UnknownError.Wrap(err))
		case gosh.NoSuchCwdError:
			panic(executor.UnknownError.Wrap(err))
		case gosh.ProcMonitorError:
			panic(executor.TaskExecError.Wrap(err))
		default:
			panic(executor.UnknownError.Wrap(err))
		}
	}).Done()

	// Wait for the job to complete
	// REVIEW: consider exposing `gosh.Proc`'s interface as part of repeatr's job tracking api?
	result.ExitCode = proc.GetExitCode()

	// Horrifyingly ambiguous attempts to detect failure modes from inside nsinit.
	// This can only be made correct by pushing patches into nsinit to use another channel for control data reporting that is completely separated from user data flows.
	// (Or, arguably, putting another layer of control processes as the first parent inside nsinit, but that's ducktape within a ducktape mesh; let's not.)
	// Certain program outputs may be incorrectly attributed as launch failure, though this should be... "unlikely".
	// Also note that if we ever switch to non-blocking execution, this will become even more of a mess: we won't be able to tell if exec failed, esp. in the case of e.g. a long running process with no output, and so we won't know when it's safe to return.

	// TODO handle the following leading strings:
	// - "exec: \"%s\": executable file not found in $PATH\n"
	// - "no such file or directory\n"
	// this will probably require rejiggering a whole bunch of stuff so that the streamer is reachable down here.

	// Save outputs
	result.Outputs = util.PreserveOutputs(transmat, f.Outputs, rootfs, journal)
}
