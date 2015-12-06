package def

/*
	Action describes the computation to be run once the inputs have been set up.
	All content is part of the conjecture.
*/
type Action struct {
	Entrypoint []string          `json:"command,omitempty"` // executable to invoke as the task.  included in the conjecture.
	Cwd        string            `json:"cwd,omitempty"`     // working directory to set when invoking the executable.  if not set, will be defaulted to "/".
	Env        map[string]string `json:"env,omitempty"`     // environment variables.  included in the conjecture.
	Escapes    Escapes           `json:"escapes,omitempty"`
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
	Mounts []Mount
}

type Mount struct {
	SourcePath string
	TargetPath string
	Writable   bool
	// CONSIDER: not sure what should be default for writable.  ro should usually be a scan; but not required; you might have sockets, be using this as ipc, whatever; all of which are "crazy", relatively speaking, but that's what this is called an escape valve for.
}
