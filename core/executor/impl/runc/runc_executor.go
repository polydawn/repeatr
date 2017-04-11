package runc

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/polydawn/gosh"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/assets"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/core/executor/basicjob"
	"go.polydawn.net/repeatr/core/executor/cradle"
	"go.polydawn.net/repeatr/core/executor/util"
	"go.polydawn.net/repeatr/lib/flak"
	"go.polydawn.net/repeatr/lib/streamer"
)

// interface assertion
var _ executor.Executor = &Executor{}

type Executor struct {
	workspacePath string
}

func (e *Executor) Configure(workspacePath string) {
	e.workspacePath = workspacePath
}

func (e *Executor) Start(f def.Formula, id executor.JobID, stdin io.Reader, log log15.Logger) executor.Job {
	// TODO this function sig and its interface are long overdue for an aggressive refactor.
	// - `journal` is Rong.  The streams mux should be accessible after this function's scope!
	//   - either that or it's time to get cracking on saving the stream mux as an output
	// - `journal` should still be a thing, but it should be a logger.
	// - All these other values should move along in a `Job` struct
	//   - `BasicJob` sorta started, but is drunk:
	//      - if we're gonna have that, it's incomplete on the inputs
	//      - for some reason it mixes in responsibility for waiting for some of the ouputs
	//      - that use of channels and public fields is stupidly indefensive
	//   - The current `Job` interface is in the wrong package
	// - almost all of the scopes in these functions is wrong
	//   - they should be realigned until they actually assist the defers and cleanups
	//     - e.g. withErrorCapture, withJobWorkPath, withFilesystems, etc

	// Fill in default config for anything still blank.
	f = *cradle.ApplyDefaults(&f)

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

			job.Result = e.Run(f, job, dir, stdin, outS, errS, log)
		}, e.workspacePath, "job", string(job.Id()))

		// Directory is clean; job complete
		close(job.WaitChan)
	}()

	<-jobReady
	return job
}

// Executes a job, catching any panics.
func (e *Executor) Run(f def.Formula, j executor.Job, d string, stdin io.Reader, outS, errS io.WriteCloser, journal log15.Logger) executor.JobResult {
	r := executor.JobResult{
		ID:       j.Id(),
		ExitCode: -1,
	}

	r.Error = meep.RecoverPanics(func() {
		e.Execute(f, j, d, &r, stdin, outS, errS, journal)
	})
	return r
}

// Execute a formula in a specified directory. MAY PANIC.
func (e *Executor) Execute(formula def.Formula, job executor.Job, jobPath string, result *executor.JobResult, stdin io.Reader, stdout, stderr io.WriteCloser, journal log15.Logger) {
	rootfsPath := filepath.Join(jobPath, "rootfs")

	// Prepare inputs
	transmat := util.DefaultTransmat()
	inputArenas := util.ProvisionInputs(transmat, formula.Inputs, journal)
	util.ProvisionOutputs(formula.Outputs, rootfsPath, journal)

	// Assemble filesystem
	assembly := util.AssembleFilesystem(
		util.BestAssembler(),
		rootfsPath,
		formula.Inputs,
		inputArenas,
		formula.Action.Escapes.Mounts,
		journal,
	)
	defer assembly.Teardown()
	if formula.Action.Cradle == nil || *(formula.Action.Cradle) == true {
		cradle.MakeCradle(rootfsPath, formula)
	}

	// Emit config for runc.
	runcConfigJsonPath := filepath.Join(jobPath, "config.json")
	cfg := EmitRuncConfigStruct(formula, job, rootfsPath, stdin != nil)
	buf, err := json.Marshal(cfg)
	if err != nil {
		panic(executor.UnknownError.Wrap(err))
	}
	ioutil.WriteFile(runcConfigJsonPath, buf, 0600)

	// Routing logs through a fifo appears to work, but we're going to use a file as a buffer anyway:
	//  in the event of nasty breakdowns, it's preferable that the runc log remain readable even if repeatr was the process to end first.
	logPath := filepath.Join(jobPath, "runc-debug.log")

	// Get handle to invokable runc plugin.
	runcPath := filepath.Join(assets.Get("runc"), "runc")

	// Prepare command to exec
	args := []string{
		"--root", filepath.Join(e.workspacePath, "shared"), // a tmpfs would be appropriate
		"--log", logPath,
		"--log-format", "json",
		"run",
		"--bundle", jobPath,
		string(job.Id()),
	}
	cmd := exec.Command(runcPath, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// launch execution.
	// transform gosh's typed errors to repeatr's hierarchical errors.
	// this is... not untroubled code: since we're invoking a helper that's then
	//  proxying the exec even further, most errors are fatal (the mapping here is
	//   very different than in e.g. chroot executor, and provides much less meaning).
	startedExec := time.Now()
	journal.Info("Beginning execution!")
	var proc gosh.Proc
	meep.Try(func() {
		proc = gosh.ExecProcCmd(cmd)
	}, meep.TryPlan{
		{ByType: gosh.NoSuchCommandError{}, Handler: func(err error) {
			panic(executor.ConfigError.New("runc binary is missing"))
		}},
		{ByType: gosh.NoArgumentsError{}, Handler: func(err error) {
			panic(executor.UnknownError.Wrap(err))
		}},
		{ByType: gosh.NoSuchCwdError{}, Handler: func(err error) {
			panic(executor.UnknownError.Wrap(err))
		}},
		{ByType: gosh.ProcMonitorError{}, Handler: func(err error) {
			panic(executor.TaskExecError.Wrap(err))
		}},
		{CatchAny: true, Handler: func(err error) {
			panic(executor.UnknownError.Wrap(err))
		}},
	})

	var runcLog io.ReadCloser
	runcLog, err = os.OpenFile(logPath, os.O_CREATE|os.O_RDONLY, 0644)
	// note this open races child; doesn't matter.
	if err != nil {
		panic(executor.TaskExecError.New("failed to tail runc log: %s", err))
	}
	// swaddle the file in userland-interruptable reader;
	//  obviously we don't want to stop watching the logs when we hit the end of the still-growing file.
	runcLog = streamer.NewTailReader(runcLog)

	// Proxy runc's logs out in realtime; also, detect errors and exit statuses from the stream.
	var realError error
	var someError bool // see the "NOTE WELL" section below -.-
	var tailerDone sync.WaitGroup
	tailerDone.Add(1)
	go func() {
		defer tailerDone.Done()
		dec := json.NewDecoder(runcLog)
		for {
			// Parse log lines.
			var logMsg map[string]string
			err := dec.Decode(&logMsg)
			if err != nil {
				if err == io.EOF {
					return
				}
				panic(executor.TaskExecError.New("unparsable log from runc: %s", err))
			}
			// remap
			if _, ok := logMsg["msg"]; !ok {
				logMsg["msg"] = ""
			}
			ctx := log15.Ctx{}
			for k, v := range logMsg {
				if k == "msg" {
					continue
				}
				ctx["runc-"+k] = v
			}

			//fmt.Printf("\n\n---\n%s\n---\n\n", logMsg["msg"])

			// Attempt to filter and normalize errors.
			// We want to be clear in representing which category of errors are coming up:
			//
			//  - Type 1.a: Exit codes of the contained user process.
			//    - These aren't errors that we raise as such: they're just an int code to report.
			//  - Type 1.b: Errors from invalid user configuration (e.g. no such executable, which prevents the process from ever starting) (we expect these to be reproducible!).
			//    - These kinds of errors should be mapped onto clear types themselves: we want a "NoSuchCwdError", not just a string vomit.
			//  - Type 2: Errors from runc being unable to function (e.g. maybe your kernel doesn't support cgroups, or other bizarre and serious issue?), where hopefully we can advise the user of this in a clear fashion.
			//  - Type 3: Runc crashing in an unrecognized way (which should result in either patches to our recognizers, or bugs filed upstream to runc).
			//
			// This is HARD.
			//
			// NOTE WELL: we cannot guarantee to capture all semantic runc failure modes.
			//  Errors may slip through with exit status 1: there are still many fail states
			//  which runc does not log with sufficient consistency or a sufficiently separate
			//  control channel for us to be able to reliably disambiguate them from stderr
			//  output of a successfully executing job!
			//
			// We have whitelisted recognizers for what we can, but oddities may remain.
			for _, tr := range []struct {
				prefix, suffix string
				err            error
			}{
				{"container_linux.go:262: starting container process caused \"exec: \\\"", ": executable file not found in $PATH\"\n",
					executor.NoSuchCommandError.New("command %q not found", formula.Action.Entrypoint[0])},
				{"container_linux.go:262: starting container process caused \"exec: \\\"", ": no such file or directory\"\n",
					executor.NoSuchCommandError.New("command %q not found", formula.Action.Entrypoint[0])},
				{"container_linux.go:262: starting container process caused \"chdir to cwd (\\\"", "\\\") set in config.json failed: not a directory\"\n",
					executor.NoSuchCwdError.New("cannot set cwd to %q: no such file or directory", formula.Action.Cwd)},
				{"container_linux.go:262: starting container process caused \"chdir to cwd (\\\"", "\\\") set in config.json failed: no such file or directory\"\n",
					executor.NoSuchCwdError.New("cannot set cwd to %q: no such file or directory", formula.Action.Cwd)},
				// Note: Some other errors were previously raised in the pattern of `executor.TaskExecError.New("runc cannot operate in this environment!")`,
				// but none of these are currently here because we cachebusted our known error strings when upgrading runc.
			} {
				if !strings.HasPrefix(logMsg["msg"], tr.prefix) {
					continue
				}
				if !strings.HasSuffix(logMsg["msg"], tr.suffix) {
					continue
				}
				realError = tr.err
				break
			}

			// Log again.
			// The level of alarm we raise depends:
			//  - With runc, everything we hear is at least a warning;
			//  - If we recognized it above, it's no more than a warning;
			//  - If we *didn't* recognize and handle it explicitly, and
			//    we can see a clear indication it's fatal, then log big and red.
			if realError == nil && ctx["runc-level"] == "error" {
				journal.Error(logMsg["msg"], ctx)
				someError = true
			} else {
				journal.Warn(logMsg["msg"], ctx)
			}
		}
	}()

	// Wait for the job to complete.
	result.ExitCode = proc.GetExitCode()
	journal.Info("Execution done!",
		"elapsed", time.Now().Sub(startedExec).Seconds(),
	)
	// Tell the log tailer to drain as soon as the proc exits.
	runcLog.Close()
	// Wait for the tailer routine to drain & exit (this sync guards the err vars).
	tailerDone.Wait()

	// If we had a CnC error (rather than the real subprocess exit code):
	//  - reset code to -1 because the runc exit code wasn't really from the job command
	//  - finally, raise the error
	// FIXME we WISH we could zero the output buffers because runc pushes duplicate error messages
	//  down a channel that's indistinguishable from the application stderr... but that's tricky for several reasons:
	//  - we support streaming them out, right?
	//  - that means we'd have to have been blocking them already; we can't zero retroactively.
	//  - there's no "all clear" signal available from runc that would let us know we're clear to start flushing the stream if we blocked it.
	//  - So, we're unable to pass the executor compat tests until patches to runc clean up this behavior.
	if someError && realError == nil {
		realError = executor.UnknownError.New("runc errored in an unrecognized fashion")
	}
	if realError != nil {
		result.ExitCode = -1
		panic(realError)
	}

	// Save outputs
	result.Outputs = util.PreserveOutputs(transmat, formula.Outputs, rootfsPath, journal)
}
