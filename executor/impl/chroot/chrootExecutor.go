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
	"go.polydawn.net/repeatr/executor/mixins"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fs/osfs"
	"go.polydawn.net/rio/fsOp"
	"go.polydawn.net/rio/stitch"
)

type Executor struct {
	workspaceFs   fs.FS             // A working dir per execution will be made in here.
	assemblerTool *stitch.Assembler // Contains: unpackTool, caching cfg, and placer tools.
	packTool      rio.PackFunc
}

var _ repeatr.RunFunc = Executor{}.Run

func (cfg Executor) Run(
	ctx context.Context,
	formula api.Formula,
	input repeatr.InputControl,
	monitor repeatr.Monitor,
) (*api.RunRecord, error) {
	// Start filling out record keeping!
	//  Includes picking a random guid for the job, which we use in all temp files.
	rr := &api.RunRecord{}
	mixins.InitRunRecord(rr, formula)

	// Make work dirs.
	//  Including whole workspace dir and parents, if necessary.
	if err := fsOp.MkdirAll(osfs.New(fs.AbsolutePath{}), cfg.workspaceFs.BasePath().CoerceRelative(), 0700); err != nil {
		return nil, Errorf(repeatr.ErrLocalCacheProblem, "cannot initialize workspace dirs: %s", err)
	}
	jobPath := fs.MustRelPath(rr.Guid)
	chrootPath := jobPath.Join(fs.MustRelPath("chroot"))
	if err := cfg.workspaceFs.Mkdir(jobPath, 0700); err != nil {
		return nil, Recategorize(err, repeatr.ErrLocalCacheProblem)
	}
	if err := cfg.workspaceFs.Mkdir(chrootPath, 0755); err != nil {
		return rr, Recategorize(err, repeatr.ErrLocalCacheProblem)
	}
	chrootFs := osfs.New(cfg.workspaceFs.BasePath().Join(chrootPath))

	// Shell out to assembler.
	unpackSpecs := stitch.FormulaToUnpackSpecs(formula, api.Filter_NoMutation)
	cleanupFunc, err := cfg.assemblerTool.Run(ctx, chrootFs, unpackSpecs)
	if err != nil {
		return rr, repeatr.ReboxRioError(err)
	}
	defer func() {
		if err := cleanupFunc(); err != nil {
			// TODO log it
		}
	}()

	// Invoke containment and run!
	cmd := buildCmd(formula, chrootFs.BasePath())
	rr.ExitCode, err = runCmd(cmd)
	if err != nil {
		return rr, err
	}

	// Pack outputs.
	packSpecs := stitch.FormulaToPackSpecs(formula)
	rr.Results, err = stitch.PackMulti(ctx, cfg.packTool, chrootFs, packSpecs)
	if err != nil {
		return rr, err
	}

	// Done!
	return rr, nil
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
	// FIXME this needs boxed symlink traversal to give correct answers
	stat, err := chrootFs.LStat(fs.MustAbsolutePath(string(frm.Action.Cwd)).CoerceRelative())
	if err != nil {
		return Errorf(repeatr.ErrJobInvalid, "cwd invalid: %s", err)
	}
	if stat.Type != fs.Type_Dir {
		return Errorf(repeatr.ErrJobInvalid, "cwd invalid: path is a %s, must be dir", stat.Type)
	}

	// Check that the command exists and is executable.
	//  (If the format is not executable, that's another ball of wax, and
	//  not so simple to detect, so we don't.)
	stat, err = chrootFs.LStat(fs.MustAbsolutePath(frm.Action.Exec[0]).CoerceRelative())
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

func buildCmd(frm api.Formula, chrootPath fs.AbsolutePath) *exec.Cmd {
	cmdName := frm.Action.Exec[0]
	cmd := exec.Command(cmdName, frm.Action.Exec[1:]...)
	// TODO port policy concepts
	// userinfo := cradle.UserinfoForPolicy(f.Action.Policy)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot: chrootPath.String(),
		// TODO port policy concepts
		//Credential: &syscall.Credential{
		//	Uid: uint32(userinfo.Uid),
		//	Gid: uint32(userinfo.Gid),
		//},
	}
	cmd.Dir = string(frm.Action.Cwd)
	cmd.Env = envToSlice(frm.Action.Env)
	// TODO IO proxy wiring
	return cmd
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
