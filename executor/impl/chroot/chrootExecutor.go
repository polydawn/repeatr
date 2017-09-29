package chroot

import (
	"context"
	"os/exec"
	"syscall"

	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/repeatr/executor/mixins"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fs/osfs"
	"go.polydawn.net/rio/stitch"
)

type Executor struct {
	workspaceFs   fs.FS             // A working dir per execution will be made in here.
	assemblerTool *stitch.Assembler // Contains: unpackTool, caching cfg, and placer tools.
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
	jobPath := fs.MustRelPath(rr.Guid)
	chrootPath := jobPath.Join(fs.MustRelPath("chroot"))
	if err := cfg.workspaceFs.Mkdir(jobPath, 0700); err != nil {
		return nil, Recategorize(err, repeatr.ErrLocalCacheProblem)
	}
	if err := cfg.workspaceFs.Mkdir(chrootPath, 0755); err != nil {
		return nil, Recategorize(err, repeatr.ErrLocalCacheProblem)
	}
	chrootFs := osfs.New(cfg.workspaceFs.BasePath().Join(chrootPath))

	// Shell out to assembler.
	unpackSpecs := stitch.FormulaToUnpackTree(formula, api.Filter_NoMutation)
	cleanupFunc, err := cfg.assemblerTool.Run(ctx, chrootFs, unpackSpecs)
	if err != nil {
		return nil, repeatr.ReboxRioError(err)
	}
	defer func() {
		if err := cleanupFunc(); err != nil {
			// TODO log it
		}
	}()

	// Invoke containment and run!
	cmd := buildCmd(formula, chrootFs.BasePath())
	// TODO
	_ = cmd

	// Pack outputs.
	// TODO

	// Done!
	return nil, nil
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

func envToSlice(env map[string]string) []string {
	rv := make([]string, len(env))
	i := 0
	for k, v := range env {
		rv[i] = k + "=" + v
		i++
	}
	return rv
}
