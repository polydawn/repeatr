package runc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/go-timeless-api/rio"
	"go.polydawn.net/repeatr/executor/cradle"
	"go.polydawn.net/repeatr/executor/mixins"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fs/osfs"
	"go.polydawn.net/rio/stitch"
)

type Executor struct {
	workspaceFs   fs.FS             // A working dir per execution will be made in here.
	cmdPath       string            // Absolute path to runc binary.
	assemblerTool *stitch.Assembler // Contains: unpackTool, caching cfg, and placer tools.
	packTool      rio.PackFunc
}

func NewExecutor(
	workDir fs.AbsolutePath,
	unpackTool rio.UnpackFunc,
	packTool rio.PackFunc,
) (repeatr.RunFunc, error) {
	asm, err := stitch.NewAssembler(unpackTool)
	if err != nil {
		return nil, repeatr.ReboxRioError(err)
	}
	cmdPath, err := findRuncBinary()
	if err != nil {
		return nil, err
	}
	return Executor{
		osfs.New(workDir),
		cmdPath,
		asm,
		packTool,
	}.Run, nil
}

// Look for the runc plugin binary -- we expect it to be in a path relative
//   to our self, OR we'll take a hint from the REPEATR_PLUGINS_PATH env var.
func findRuncBinary() (string, error) {
	pluginsPath := os.Getenv("REPEATR_PLUGINS_PATH")
	if pluginsPath == "" {
		selfPath, err := os.Executable()
		if err != nil {
			return "", Errorf(repeatr.ErrExecutor, "runc executor not available: cannot find plugin: %s", err)
		}
		pluginsPath = filepath.Join(filepath.Dir(selfPath), "plugins")
	}
	expectedPath := filepath.Join(pluginsPath, "repeatr-plugin-runc")
	_, err := exec.LookPath(expectedPath)
	if err != nil {
		return "", Errorf(repeatr.ErrExecutor, "runc executor not available: cannot find plugin: %s", err)
	}
	return expectedPath, nil
}

var _ repeatr.RunFunc = Executor{}.Run

func (cfg Executor) Run(
	ctx context.Context,
	formula api.Formula,
	formulaCtx api.FormulaContext,
	input repeatr.InputControl,
	mon repeatr.Monitor,
) (*api.RunRecord, error) {
	if mon.Chan != nil {
		defer close(mon.Chan)
	}

	// Workspace setup and params defaulting.
	formula = cradle.FormulaDefaults(formula) // Initialize formula default values.
	rr := api.RunRecord{}                     // Start filling out record keeping!
	mixins.InitRunRecord(&rr, formula)        // Includes picking a random guid for the job, which we use in all temp files.

	// Make work dirs. Including whole workspace dir and parents, if necessary.
	jobFs, chrootFs, err := mixins.MakeWorkDirs(cfg.workspaceFs, rr)
	if err != nil {
		return nil, err
	}

	// Use standard filesystem setup/teardown, handing it our 'run' thunk
	//  to invoke while it's living.
	rr.Results, err = mixins.WithFilesystem(ctx,
		chrootFs, cfg.assemblerTool, cfg.packTool,
		formula, formulaCtx, mon,
		func(chrootFs fs.FS) (err error) {
			rr.ExitCode, err = cfg.run(ctx, rr.Guid, formula.Action, jobFs, chrootFs, input, mon)
			return
		},
	)
	return &rr, err
}

func (cfg Executor) run(
	ctx context.Context,
	jobID string,
	action api.FormulaAction,
	jobFs fs.FS, // a spot for other tmp/job-lifetime files.
	chrootFs fs.FS,
	input repeatr.InputControl,
	mon repeatr.Monitor,
) (int, error) {
	// Check that action commands appear to be executable on this filesystem.
	if err := mixins.CheckFSReadyForExec(action, chrootFs); err != nil {
		return -1, err
	}

	// Configure the container.
	//  For runc, this means we have to actually *write config to disk*.
	//  We'll pass that path as an arg again shortly.
	useTty := false
	if input.Chan != nil {
		useTty = true
	}
	runcCfg, err := templateRuncConfig(jobID, action, chrootFs.BasePath().String(), useTty)
	if err != nil {
		return -1, err
	}
	runcCfgPathStr := jobFs.BasePath().String() + "/config.json"
	if err := writeConfigToFile(runcCfgPathStr, runcCfg); err != nil {
		return -1, err
	}

	// Select a path for runc logs.
	//  Again, we'll need this again shortly.
	runcLogPathStr := jobFs.BasePath().String() + "/log"

	// Start templating commands.
	cmd := exec.Command(cfg.cmdPath,
		"--root", jobFs.BasePath().String()+"/tmp",
		"--debug",
		"--log", runcLogPathStr,
		"--log-format", "json",
		"run",
		"--bundle", jobFs.BasePath().String(),
		jobID,
	)

	// Wire I/O.
	if input.Chan != nil {
		// Dire hack: reach all the way out to the system stdin handle.
		// We need this for TTY reasons.
		// Future work: do our own PTY management, giving us room for handling
		//  custom escape sequences, isolating this code better, etc.
		cmd.Stdin = os.Stdin
	}
	proxy := mixins.NewOutputEventWriter(ctx, mon.Chan)
	cmd.Stdout = proxy // TODO probably more here
	cmd.Stderr = proxy // TODO probably more here

	// Launch runc process.
	if err := cmd.Start(); err != nil {
		return -1, Errorf(repeatr.ErrExecutor, "executor failed to launch: %s", err)
	}

	// Watch logs; we have additional output handling to do.
	// TODO

	// Await command completion; return its exit code.
	//  (If we get this far, the code from the 'real' work proc is all that's left.)
	return cmdWait(cmd)
}

// copypasta glue for get-the-real-exitcode-plz
func cmdWait(cmd *exec.Cmd) (int, error) {
	err := cmd.Wait()
	if err == nil {
		return 0, nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok { // This is basically an "if stdlib isn't what we thought it is" error, so panic-worthy.
		panic(fmt.Errorf("unknown exit reason: %T %s", err, err))
	}
	waitStatus, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus)
	if !ok { // This is basically a "if stdlib[...]" or OS portability issue, so also panic-able.
		panic(fmt.Errorf("unknown process state implementation %T", exitErr.ProcessState.Sys()))
	}
	if waitStatus.Exited() {
		return waitStatus.ExitStatus(), nil
	} else if waitStatus.Signaled() {
		// In bash, when a processs ends from a signal, the $? variable is set to 128+SIG.
		// We follow that same convention here.
		// So, a process terminated by ctrl-C returns 130.  A script that died to kill-9 returns 137.
		return int(waitStatus.Signal()) + 128, nil
	} else {
		return -1, Errorf(repeatr.ErrExecutor, "unknown process wait status (%#v)", waitStatus)
	}
}
