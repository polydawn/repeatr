package cradle

// these can easily operate by massaging a formula:
//   - defaultWorkingDir
//   - defaultPathEnv
//   - defaultHomeEnv

// these all return properties used by executor:
//   - capsForPolicy
//   - uidGidForPolicy
//   - dirpropsForPolicy

// all of these are both mkdir and force props:
//	 - ensureWorkingDir(rootfsPath, frm)
//	 - ensureHomeDir(rootfsPath, frm.Action.Policy)
//	 - ensureTempDir(rootfsPath)
// this also requires filesystem delta'ing, and is... hard:
//	 - ensureIdentityFiles(rootfsPath, frm.Action.Policy)

// we've punted on these so far:
//   - ensureNetworkConf

/*
	SO, design questions!

	Do we need to make these *individually* disable'able?
	(Probably not.)

	Do we need to do anything fancy with $PATH merging?
	(No.)

	That was easy.

	Do we need to do anything about network config?
	(No.  The "right thing" likely involves More Mounts, which
	we'll leave to be an executor problem (if probably a mixin),
	and also does not vary based on policy.  With one exception...)

	Should we make network isolation switches near here?
	(Not sure!  Do we consider that a "policy" thing?
	Probably not.  It's a variable independing of all the other
	security/local-privs concerns.)
*/
