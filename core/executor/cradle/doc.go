/*
	The `cradle` package defines and provide utitilies for setting up
	a standard "minimum viable environment" for a easily starting a
	contained process that has no priviledges itself.

	In the big picture: We want to target dropping as many privileges as
	we can, while simultaneously enabling as much regular, easy usage
	as possible.  Operating with the principle of least priviledge
	should be frictionless, in other words.

	Frankly, this is hard because of ancient choices (read: "mistakes" --
	even if to be fair they weren't obvious at the time) in linux, and
	before that the entire unix model.
	Very basic operations that a new user needs to do to have a minimum
	viable situation to work in require explicit gated setup by an administrator.
	So, nowadays, when developing containment systems, we not only need to
	sandbox things and drop privs, we need to -- at the same time -- deal
	with all the Little Things we need to make the guest comfortable.

	The container model of the world draws extra attention to this: and
	since your root filesystem snapshot and your actuall process-of-interest
	runtime configuration are essentially produced by different actors,
	implicity, there would have to be a loose contract between the two.
	Implicit, loose contracts are bad for reliable systems and bad for
	comprehension: so, instead, we're going to provide a reliable baseline
	so theyre's no need for that implicit handwaving in the first place.

	So, that's `cradle`'s job:
	  - Define broad easy-to-understand permission levels.
	  - Map those broad definitions onto concrete capabilities we want to drop.
	  - Ensure a filesystem that's usable: e.g., if you're not uid=0
	    and don't have CAP_FOWNER, we'd better make sure you start with a writable dir!
	  - Ensure a minimal identity: e.g., many linux programs crash outright
	    if they can't find your uid in /etc/passwd, so we'll ensure that's there.
	  - Ensure an environment that's sane: e.g., for better or worse,
	    everyone and their dog expects you to have a $HOME, so we'll do that.

	Cradle is a utility package for use by executor implementations.
*/
package cradle
