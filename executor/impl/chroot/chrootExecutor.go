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
	formula = cradle.FormulaDefaults(formula)                    // Initialize formula default values.
	rr := api.RunRecord{}                                        // Start filling out record keeping!
	mixins.InitRunRecord(&rr, formula)                           // Includes picking a random guid for the job, which we use in all temp files.
	_, chrootFs, err := mixins.MakeWorkDirs(cfg.workspaceFs, rr) // Make work dirs. Including whole workspace dir and parents, if necessary.
	if err != nil {
		return nil, err
	}

	// Shell out to assembler.
	unpackSpecs := stitch.FormulaToUnpackSpecs(formula, formulaCtx, api.Filter_NoMutation)
	wgRioLogs := mixins.ForwardRioUnpackLogs(ctx, mon, unpackSpecs)
	cleanupFunc, err := cfg.assemblerTool.Run(ctx, chrootFs, unpackSpecs, cradle.DirpropsForUserinfo(*formula.Action.Userinfo))
	wgRioLogs.Wait()
	if err != nil {
		return &rr, repeatr.ReboxRioError(err)
	}
	defer func() {
		if err := cleanupFunc(); err != nil {
			// TODO log it
		}
	}()

	// Last bit of filesystem brushup: run cradle fs mutations.
	if err := cradle.TidyFilesystem(formula, chrootFs); err != nil {
		return &rr, err
	}

	// Check the action commands can look to be executable on this filesystem.
	if err := mixins.CheckFSReadyForExec(formula, chrootFs); err != nil {
		return &rr, err
	}

	// Invoke containment and run!
	cmdName := formula.Action.Exec[0]
	cmd := exec.Command(cmdName, formula.Action.Exec[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot: chrootFs.BasePath().String(),
		Credential: &syscall.Credential{
			Uid: uint32(*formula.Action.Userinfo.Uid),
			Gid: uint32(*formula.Action.Userinfo.Gid),
		},
	}
	cmd.Dir = string(formula.Action.Cwd)
	cmd.Env = envToSlice(formula.Action.Env)
	if input.Chan != nil {
		pipe, _ := cmd.StdinPipe()
		go func() {
			for {
				chunk, ok := <-input.Chan
				if !ok {
					pipe.Close()
					return
				}
				pipe.Write([]byte(chunk))
			}
		}()
	}
	proxy := mixins.NewOutputEventWriter(ctx, mon.Chan)
	cmd.Stdout = proxy
	cmd.Stderr = proxy
	rr.ExitCode, err = runCmd(cmd)
	if err != nil {
		return &rr, err
	}

	// Pack outputs.
	packSpecs := stitch.FormulaToPackSpecs(formula, formulaCtx, api.Filter_DefaultFlatten)
	rr.Results, err = stitch.PackMulti(ctx, cfg.packTool, chrootFs, packSpecs)
	if err != nil {
		return &rr, err
	}

	// Done!
	return &rr, nil
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
