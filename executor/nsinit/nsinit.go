package nsinit

import (
	"io"
	"os/exec"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/basicjob"
	"polydawn.net/repeatr/executor/util"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/lib/streamer"
	"polydawn.net/repeatr/output"
)

// interface assertion
var _ executor.Executor = &Executor{}

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
func (e *Executor) Execute(f def.Formula, j def.Job, d string, result *def.JobResult, outS, errS io.WriteCloser, journal io.Writer) {
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

	// Add all desired environment variables
	for k, v := range f.Accents.Env {
		args = append(args, "--env", k+"="+v)
	}

	// Unroll command args
	args = append(args, f.Accents.Entrypoint...)

	// Prepare command to exec
	cmd := exec.Command("nsinit", args...)

	cmd.Stdin = nil
	cmd.Stdout = outS
	cmd.Stderr = errS

	// Prepare filesystem
	util.ProvisionInputs(f.Inputs, rootfs, journal)
	util.ProvisionOutputs(f.Outputs, rootfs, journal)

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	// Save outputs
	result.Outputs = util.PreserveOutputs(f.Outputs, rootfs, journal)
}
