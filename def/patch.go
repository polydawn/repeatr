package def

/*
	Apply another formula as a patch to this one.

	Mutates the original (and also return a reference for convenient chaining).

	Not all fields can be merged or patched.  The current set of allowed fields is:

	  - Env vars.  (*New* values will override existing values.)

	Where exposed, this feature makes it possible to sneak in last-stage config
	values (without reading a full formula in, mutating it, and emitting JSON again).
	This feature is only exposed in some commands (and in particular, will *never*
	be supported in pipelines, because it simply wouldn't make any sense) because
	it tends to be counter-productive for clear reproducible processes, but if
	you're using repeatr as the shipping mechanism and runtime environment for
	a daemon of some kind, or doing development iteration/testing with various values
	for some feature, this may be quite handy.

*/
func (f *Formula) ApplyPatch(f2 Formula) *Formula {
	f.Action.Env = f2.Action.Env.Clone().Merge(f.Action.Env)
	// future: `f.Action.Escapes.Mounts` would make sense as well.
	return f
}
