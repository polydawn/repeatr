package chroot

import (
	"context"
	"fmt"
	"os/exec"
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
	return Executor{
		osfs.New(workDir),
		asm,
		packTool,
	}.Run, nil
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
	_, chrootFs, err := mixins.MakeWorkDirs(cfg.workspaceFs, rr)
	if err != nil {
		return nil, err
	}

	// Use standard filesystem setup/teardown, handing it our 'run' thunk
	//  to invoke while it's living.
	rr.Results, err = mixins.WithFilesystem(ctx,
		chrootFs, cfg.assemblerTool, cfg.packTool,
		formula, formulaCtx, mon,
		func(chrootFs fs.FS) (err error) {
			rr.ExitCode, err = run(ctx, formula.Action, chrootFs, input, mon)
			return
		},
	)
	return &rr, err
}

func run(
	ctx context.Context,
	action api.FormulaAction,
	chrootFs fs.FS,
	input repeatr.InputControl,
	mon repeatr.Monitor,
) (int, error) {
	// Check that action commands appear to be executable on this filesystem.
	if err := mixins.CheckFSReadyForExec(action, chrootFs); err != nil {
		return -1, err
	}

	// Configure the container.
	cmdName := action.Exec[0]
	cmd := exec.Command(cmdName, action.Exec[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot: chrootFs.BasePath().String(),
		Credential: &syscall.Credential{
			Uid: uint32(*action.Userinfo.Uid),
			Gid: uint32(*action.Userinfo.Gid),
		},
	}
	cmd.Dir = string(action.Cwd)
	cmd.Env = envToSlice(action.Env)

	// Wire I/O.
	if input.Chan != nil {
		pipe, _ := cmd.StdinPipe()
		mixins.RunInputWriteForwarder(ctx, pipe, input.Chan)
	}
	proxy := mixins.NewOutputEventWriter(ctx, mon.Chan)
	cmd.Stdout = proxy
	cmd.Stderr = proxy

	// Invoke!
	return runCmd(cmd)
}

func runCmd(cmd *exec.Cmd) (int, error) {
	if err := cmd.Start(); err != nil {
		return -1, Errorf(repeatr.ErrExecutor, "executor failed to launch: %s", err)
	}
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

func envToSlice(env map[string]string) []string {
	rv := make([]string, len(env))
	i := 0
	for k, v := range env {
		rv[i] = k + "=" + v
		i++
	}
	return rv
}
