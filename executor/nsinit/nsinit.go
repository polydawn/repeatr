package nsinit

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/basicjob"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/lib/flak"
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

	return def.Job(job)
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

	// For now, run in this terminal
	cmd := exec.Command("nsinit", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Prepare filesystem
	flak.ProvisionInputs(f.Inputs, rootfs)
	flak.ProvisionOutputs(f.Outputs, rootfs)

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	// Save outputs
	result.Outputs = flak.PreserveOutputs(f.Outputs, rootfs)
	return result
}
