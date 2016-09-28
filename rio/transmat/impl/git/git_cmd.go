package git

import "github.com/polydawn/gosh"

func bakeGitDir(cmd gosh.Command, gitDir string) gosh.Command {
	return cmd.Bake(
		gosh.Opts{Env: map[string]string{
			"GIT_DIR": gitDir,
		}},
	)
}

func bakeCheckoutDir(cmd gosh.Command, workDir string) gosh.Command {
	return cmd.Bake(gosh.Opts{
		Env: map[string]string{"GIT_WORK_TREE": workDir},
		Cwd: workDir,
	})
}

const (
	git_uid = 1000
	git_gid = 1000
)

var git gosh.Command = gosh.Gosh(
	"git",
	gosh.NullIO,
	gosh.Opts{
		Env: map[string]string{
			"GIT_CONFIG_NOSYSTEM": "true",
			"HOME":                "/dev/null",
			"GIT_ASKPASS":         "/bin/true",
		},
		// We would *LOVE* to uncomment this block and drop privs.
		// However, it's currently practically un-supportable: git running
		// on a host that doesn't contain a username mapped to this uid
		// will error on launch -- yep, it's one of *those* programs
		// (at least as of 1.9.1; more recent upstreams *may* have patched it;
		// haven't tested exhaustively yet.)
		// To address this, we'd either need containerized-git (which may
		// limit portability in some other undesirable fashions; ideally
		// transmats should work without such heavy weaponry), or distributing
		// a reference a particular (and likely patched) version of git.
		// ----------------------------------------------------------------
		//	Launcher: gosh.ExecCustomizingLauncher(func(cmd *exec.Cmd) {
		//		cmd.SysProcAttr = &syscall.SysProcAttr{
		//			Credential: &syscall.Credential{
		//				Uid: uint32(git_uid),
		//				Gid: uint32(git_gid),
		//			},
		//		}
		//	}),
	},
)
