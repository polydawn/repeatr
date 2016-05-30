package def

/*
	Action describes the computation to be run once the inputs have been set up.
	All content is part of the conjecture.
*/
type Action struct {
	Entrypoint []string `json:"command,omitempty"` // executable to invoke as the task.  included in the conjecture.
	Cwd        string   `json:"cwd,omitempty"`     // working directory to set when invoking the executable.  if not set, will be defaulted to "/".
	Env        Env      `json:"env,omitempty"`     // environment variables.  included in the conjecture.
	Policy     Policy   `json:"policy,omitempty"`  // policy naming user level and security mode.
	Cradle     *bool    `json:"cradle,omitempty"`  // default/nil interpreted as true; set to false to disable ensuring cradle during setup.
	Escapes    Escapes  `json:"escapes,omitempty"`
}

type Env map[string]string

/*
	`Policy` constants enumerate the priviledge levels and default situation
	to start a contained process in.

	By default, repeatr will try to use at least as much isolation as a
	regular posix user account would provide, and will automatically
	provision that account.
	You can configure other policies to retain more priviledges (though
	this may mean correspondingly decreased security, and you should read
	the documentation on each Policy before using it).

	Policies are meant as a rough, relatively approachable, user-facing outline.
	Different executors may implement policies differently.  All executors
	MUST assign a non-zero UID to implement the `Routine` Policy.
	Other levels may be less clearly defined.  (Specifically, the `chroot`
	executor simply cannot provide fine-grained capabilities, and is
	significantly insecure in general; you have been warned.)
	If you need extremely specific control over e.g. assignment of
	specific linux capability bitsets, you'll need to use other overrides.
	Policies are meant for every-day ease of use, not all possible situations.

	Mostly, Policy is about how much priviledge your process starts with,
	but it also has some impact on system setup: specifically, if you
	use the default mode, which gives you a regular user account, the
	executor will modify your filesystem image to provide that account
	(see the `executor/cradle` package for more about this behavior).
*/
type Policy string

var PolicyValues = []Policy{
	PolicyRoutine,
	PolicyUidZero,
	PolicyGovernor,
	PolicySysad,
}

const (
	/*
		Operate with a low uid, as if you were a regular user on a
		regular system.  No special permissions will be granted
		(and in systems with capabilities support, special permissions
		will not be available even if processes do manage to
		change uid, e.g. through suid binaries; most capabilities
		are dropped).

		The default uid to switch to is uid=1000,gid=1000.
		These are same as the default filters applied to file permissions,
		which should usually result in seamless "do the right thing".

		This is the safest mode to run as.  And, naturally, the default.
	*/
	PolicyRoutine = Policy("routine") // routine?  regular?  ordinary?  user?

	/*
		Operate with uid=0, but drop all interesting capabilities.
		This means things root would normally be able to do (like chown
		any file) will result in permission denied.

		Use this when you need uid=0 for some reason, but don't actually
		require any root-like priviledges.
		This can be useful for tricking programs that *think* they
		need to be uid=0, but really don't actually require very special
		priviledges during their work (hey, almost everyone is guilty
		of having written shell scripts that do a quick-and-dirty
		`$UID -eq 0` check before we knew better).

		It may also be useful for running tools which need to modify
		files owned by uid=0, but don't otherwise require special
		priviledges (`apt` tools are frequently an example of this).
		However, consider configuring `Filter`s in your formulas to
		manage filesystem permissions out-of-band instead.

		Usually if you can use 'uidzero', you can go all the way down to
		'routine' mode (and you should, it's the default after all -- one
		less thing to configure!).  Try using `Filter`s to close the
		file permissions gap if you have one, or try patching away
		the problematic uid checks, etc.
	*/
	PolicyUidZero = Policy("uidzero")

	/*
		Operate with uid=0, with some of the most dangers capabilities
		(e.g. "muck with devices") dropped, but most of root's powers
		(like chown any file) still available.

		This may be slightly safer than enabling full 'sysad' mode,
		but you should still prefer to use any of the lower power levels
		if possible.

		Note that even if you drop to a non-zero uid inside your process
		tree, setuid binaries will still work as normal: other processes
		may be able to re-escalate.  Including e.g. a password-based
		sudo system in your filesystem image is probably not wise.

		This mode is the most similar to what you would experience with
		docker defaults.
	*/
	PolicyGovernor = Policy("governor")

	/*
		Operate with uid=0 and *ALL CAPABILITIES*.

		This is absolutly not secure against untrusted code -- it is
		completely equivalent in power to root on your host.  Please
		try to use any of the lower power levels first.

		Among the things a system administrator may do is rebooting
		the machine and updating the kernel.  Seriously, *only* use
		with trusted code.
	*/
	PolicySysad = Policy("sysad")
)

/*
	Escapes are features that give up repeatr's promises about repeatability,
	but make it possible to use repeatr's execution engines and data transports
	anyway.

	For example, one "escape" is to make a writable mount of a host
	filesystem.  This instantly breaks all portability guarantees... but is
	incredibly useful if you want to use repeatr inputs to ship an application
	that is then allowed to interact statefully with a host machine.

	If you want to use data from a host machine, but still want trackable
	repeatability guarantees, consider using a pipeline instead of host mounts:
	use `repeatr scan --type=dir` to hash the data (and optionally store copies),
	then pipe the hash reported by the scan into the formula you hand to `repeatr run`.
*/
type Escapes struct {
	Mounts MountGroup `json:"mounts,omitempty"`
}

type MountGroup []Mount

type Mount struct {
	TargetPath string
	SourcePath string
	Writable   bool // defaults to false.  if you forget a conf word -> fail safe.
}
