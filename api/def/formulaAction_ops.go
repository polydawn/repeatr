package def

func (a Action) Clone() Action {
	cpyEntrypoint := make([]string, len(a.Entrypoint))
	copy(cpyEntrypoint, a.Entrypoint)
	a.Entrypoint = cpyEntrypoint
	a.Env = a.Env.Clone()
	// punt on escapes, still haven't seen an excuse to mutate
	return a
}
func (e Env) Clone() Env {
	r := make(Env, len(e))
	for k, v := range e {
		r[k] = v
	}
	return r
}

/*
	Merge given env map into the object.
	Existing values are preferred, new values are added.
	Mutates; `Clone()` first to avoid if necessary.
	Returns the mutated reference for convenient chaining.
*/
func (keep Env) Merge(other Env) Env {
	for k, v := range other {
		if _, ok := keep[k]; !ok {
			keep[k] = v
		}
	}
	return keep
}

type MountGroupByTargetPath MountGroup

func (a MountGroupByTargetPath) Len() int           { return len(a) }
func (a MountGroupByTargetPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a MountGroupByTargetPath) Less(i, j int) bool { return a[i].TargetPath < a[j].TargetPath }
