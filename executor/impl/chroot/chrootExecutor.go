package chroot

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
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

	// Start filling out record keeping!
	//  Includes picking a random guid for the job, which we use in all temp files.
	rr := api.RunRecord{}
	mixins.InitRunRecord(&rr, formula)

	// Make work dirs.
	//  Including whole workspace dir and parents, if necessary.
	_, chrootFs, err := mixins.MakeWorkDirs(cfg.workspaceFs, rr)

	// Initialize default values.
	formula = cradle.FormulaDefaults(formula)

	// Shell out to assembler.
	unpackSpecs := stitch.FormulaToUnpackSpecs(formula, formulaCtx, api.Filter_NoMutation)
	var wg sync.WaitGroup
	if mon.Chan != nil {
		for i, _ := range unpackSpecs {
			wg.Add(1)
			ch := make(chan rio.Event)
			unpackSpecs[i].Monitor = rio.Monitor{ch}
			go func() {
				defer wg.Done()
				for {
					select {
					case evt, ok := <-ch:
						if !ok {
							return
						}
						switch {
						case evt.Log != nil:
							mon.Chan <- repeatr.Event{Log: &repeatr.Event_Log{
								Time:   evt.Log.Time,
								Level:  repeatr.LogLevel(evt.Log.Level),
								Msg:    evt.Log.Msg,
								Detail: evt.Log.Detail,
							}}
						case evt.Progress != nil:
							// pass... for now
						}
					case <-ctx.Done():
						return
					}
				}
			}()
		}
	}
	cleanupFunc, err := cfg.assemblerTool.Run(ctx, chrootFs, unpackSpecs, cradle.DirpropsForUserinfo(*formula.Action.Userinfo))
	wg.Wait()
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

	// Sanity check the ready filesystem.
	//  Some errors produce *very* unclear results from exec (for example
	//  at the kernel level, EACCES can mean *many* different things...), and
	//  so it's better that we try to detect common errors early and thus be
	//  able to give good messages.
	if err := sanityCheckFs(formula, chrootFs); err != nil {
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

/*
	Return an error if any part of the filesystem is invalid for running the
	formula -- e.g. the CWD setting isn't a dir; the command binary
	does not exist or is not executable; etc.

	The formula is already expected to have been syntactically validated --
	e.g. all paths have been checked to be absolute, etc.  This method will
	panic if such invarients aren't held.

	(It's better to check all these things before attempting to launch
	containment because the error codes returned by kernel exec are sometimes
	remarkably ambiguous or outright misleading in their names.)

	Currently, we require exec paths to be absolute.
*/
func sanityCheckFs(frm api.Formula, chrootFs fs.FS) error {
	// Check that the CWD exists and is a directory.
	stat, err := chrootFs.Stat(fs.MustAbsolutePath(string(frm.Action.Cwd)).CoerceRelative())
	if err != nil {
		return Errorf(repeatr.ErrJobInvalid, "cwd invalid: %s", err)
	}
	if stat.Type != fs.Type_Dir {
		return Errorf(repeatr.ErrJobInvalid, "cwd invalid: path is a %s, must be dir", stat.Type)
	}

	// Check that the command exists and is executable.
	//  (If the format is not executable, that's another ball of wax, and
	//  not so simple to detect, so we don't.)
	stat, err = chrootFs.Stat(fs.MustAbsolutePath(frm.Action.Exec[0]).CoerceRelative())
	if err != nil {
		return Errorf(repeatr.ErrJobInvalid, "exec invalid: %s", err)
	}
	if stat.Type != fs.Type_File {
		return Errorf(repeatr.ErrJobInvalid, "exec invalid: path is a %s, must be executable file", stat.Type)
	}
	// FUTURE: ideally we could also check if the file is properly executable,
	//  and all parents have bits to be traversable (!), to the policy uid.
	//  But this is also a loooot of work: and a correct answer (for groups
	//  at least) requires *understanding the container's groups settings*,
	//  and now you're in real hot water: parsing /etc files and hoping
	//  nobody expects nsswitch to be too interesting.  Yeah.  Nuh uh.
	//  (All of these are edge conditions tools like docker Don't Have because
	//  they simply launch you with so much privilege that it doesn't matter.)

	return nil
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
