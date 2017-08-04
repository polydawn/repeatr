package repeatr

/*
	A brief overview of what different implementations contend with:

	The API at large:

		- has `Run(ctx, frm, disco, <-watch) (runrecord, err)` method.
		  - The amount of stuff you had to factory to *get* that is not my problem.

	Implementations vary wildly in the kind of setup they need to consume before
	being able to make that method fly:

		- The regular runs-on-your-host repeatr CLI:
		  - We need the following setup:
		    - workspace paths
		    - constraints on executors
		    - transmat plugins
		    - constraints on fileset assemblers
		  - When running from our own CLI, we tend to pick up the 'disco' helper info from our own cwd and environment.
		    - Not yet seen: "workspace" config, which *is* in fact diff than hitch: it's *yours* (much like .git/config urls vs the committed ones).
		    - So in this one case: the entry point (the CLI) has influence over *both* setup and the `Run()` args.  This is not the norm; don't get distracted by it.
		- Proxied by r2k8s:
		  - We need the following setup:
		    - a k8s config file, dial info, and auth tokens.
		    - maybe some mapping info on things we're allowed to automagically hoistmirror?
		  - You actually lose the ability to specify some things:
		    - basically everything.  it's the cluster's decision.
		- A mock version:
		  - We need the following setup:
		    - *nothing* because we're going to only follow certain very silly kinds of instructions.
*/
