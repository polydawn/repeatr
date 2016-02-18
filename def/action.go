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

func (a Action) Clone() Action {
	cpyEntrypoint := make([]string, len(a.Entrypoint))
	copy(cpyEntrypoint, a.Entrypoint)
	a.Entrypoint = cpyEntrypoint
	a.Env = a.Env.Clone()
	// punt on escapes, still haven't seen an excuse to mutate
	return a
}

type Env map[string]string

func (e Env) Clone() Env {
	r := make(Env, len(e))
	for k, v := range e {
		r[k] = v
	}
	return r
}

/*
	Merge given env map into the object.
	Existing values are preferred,	new values are added.
	Mutates; `Clone()` first to avoid if necessary.
*/
func (keep Env) Merge(other Env) {
	for k, v := range other {
		if _, ok := keep[k]; !ok {
			keep[k] = v
		}
	}
}

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

type MountGroupByTargetPath MountGroup

func (a MountGroupByTargetPath) Len() int           { return len(a) }
func (a MountGroupByTargetPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a MountGroupByTargetPath) Less(i, j int) bool { return a[i].TargetPath < a[j].TargetPath }
