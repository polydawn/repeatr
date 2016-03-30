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

	"github.com/inconshreveable/log15"
	"github.com/polydawn/gosh"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/core/executor/basicjob"
	"polydawn.net/repeatr/core/executor/cradle"
	"polydawn.net/repeatr/core/executor/util"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/lib/streamer"
	"polydawn.net/repeatr/rio"
	"polydawn.net/repeatr/rio/assets"
)

// interface assertion
var _ executor.Executor = &Executor{}

type Executor struct {
	workspacePath string
}

func (e *Executor) Configure(workspacePath string) {
	e.workspacePath = workspacePath
}

func (e *Executor) Start(f def.Formula, id def.JobID, stdin io.Reader, journal io.Writer) def.Job {
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
		e.Execute(f, j, d, &r, stdin, outS, errS, journal)
	}).Catch(executor.Error, func(err *errors.Error) {
		r.Error = err
	}).Catch(rio.Error, func(err *errors.Error) {
		r.Error = err
	}).CatchAll(func(err error) {
		r.Error = executor.UnknownError.Wrap(err).(*errors.Error)
	}).Done()

	return r
}

// Execute a formula in a specified directory. MAY PANIC.
func (e *Executor) Execute(formula def.Formula, job def.Job, jobPath string, result *def.JobResult, stdin io.Reader, stdout, stderr io.WriteCloser, journal log15.Logger) {
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

	// Emit configs for runc.
	runcConfigJsonPath := filepath.Join(jobPath, "config.json")
	cfg := EmitRuncConfigStruct(formula, job, rootfsPath, stdin != nil)
	buf, err := json.Marshal(cfg)
	if err != nil {
		panic(executor.UnknownError.Wrap(err))
	}
	ioutil.WriteFile(runcConfigJsonPath, buf, 0600)
	runcRuntimeJsonPath := filepath.Join(jobPath, "runtime.json")
	cfg = EmitRuncRuntimeStruct(formula)
	buf, err = json.Marshal(cfg)
	if err != nil {
		panic(executor.UnknownError.Wrap(err))
	}
	ioutil.WriteFile(runcRuntimeJsonPath, buf, 0600)

	// Routing logs through a fifo appears to work, but we're going to use a file as a buffer anyway:
	//  in the event of nasty breakdowns, it's preferable that the runc log remain readable even if repeatr was the process to end first.
	logPath := filepath.Join(jobPath, "runc-debug.log")

	// Get handle to invokable runc plugin.
	runcPath := filepath.Join(assets.Get("runc"), "bin/runc")

	// Prepare command to exec
	args := []string{
		"--id", string(job.Id()),
		"--root", filepath.Join(e.workspacePath, "shared"), // a tmpfs would be appropriate
		"--log", logPath,
		"--log-format", "json",
		"start",
		"--config-file", runcConfigJsonPath,
		"--runtime-file", runcRuntimeJsonPath,
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
	var proc gosh.Proc
	try.Do(func() {
		proc = gosh.ExecProcCmd(cmd)
	}).CatchAll(func(err error) {
		switch err.(type) {
		case gosh.NoSuchCommandError:
			panic(executor.ConfigError.New("runc binary is missing"))
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
			// with runc, everything we hear is at least a warning.
			journal.Warn(logMsg["msg"], ctx)
			// actually filtering the interesting structures and raising issues
			// note that we don't need to capture the "exit status" message, because that
			//  code *does* come out correctly... but we do need to sometimes override it again.
			// NOTE WELL: we cannot guarantee to capture all semantic runc failure modes.
			//  Errors may slip through with exit status 1: there are still many fail states
			//  which runc does not log with sufficient consistency or a sufficiently separate
			//  control channel for us to be able to reliably disambiguate them from stderr
			//  output of a successfully executing job!
			// We have whitelisted what we can; the following oddities remain:
			//   - runc will log an "exit status ${n}" message for other failures of its internal forking
			//     - this one is at least on a clear control channel, so we raise it as a panic, even if we don't know what it is
			//   - lots of system initialization paths in runc will error directly stderr with no clear sigils or separation from usermode stderr.
			//     - and these mean we're just screwed, and require additional upstream patches to address.
			switch logMsg["msg"] {
			case "Container start failed: [8] System error: no such file or directory":
				realError = executor.NoSuchCwdError.New("cannot set cwd to %q: no such file or directory", formula.Action.Cwd)
			case "Container start failed: [8] System error: not a directory":
				realError = executor.NoSuchCwdError.New("cannot set cwd to %q: not a directory", formula.Action.Cwd)
			case "Container start failed: [8] System error: fork/exec /proc/self/exe: invalid argument":
				realError = executor.TaskExecError.New("runc cannot operate in this environment!  %s", "fork/exec /proc/self/exe: invalid argument")
			default:
				// broader patterns required for some of these so we can ignore the vagaries of how the command name was quoted
				if strings.HasPrefix(logMsg["msg"], "Container start failed: [8] System error: exec: ") {
					if strings.HasSuffix(logMsg["msg"], ": executable file not found in $PATH") {
						realError = executor.NoSuchCommandError.New("command %q not found", formula.Action.Entrypoint[0])
					} else if strings.HasSuffix(logMsg["msg"], ": no such file or directory") {
						realError = executor.NoSuchCommandError.New("command %q not found", formula.Action.Entrypoint[0])
					}
				} else if strings.HasPrefix(logMsg["msg"], "exit status ") {
					// by itself this doesn't mean much, but if we can't figure out
					//  what happened in particular by the proc exit, be quite alarmed.
					// and no, this does *not* mean the real job process exit code, as you might expect.
					someError = true
				}
			}
		}
	}()

	// Wait for the job to complete.
	// Tell the log tailer to drain as soon as the proc exits.
	result.ExitCode = proc.GetExitCode()
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
