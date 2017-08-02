package core

import (
	"go.polydawn.net/repeatr/executor"
)

type Runner struct {
	Executor  executor.Interface // if this was picked above, someone must've already done caps checks
	Discovery interface{}        // oh boy
	RioSvc    interface{}        // help
}

/*
	Things that this method will do:

		- Everything fun.  That means:
			- Conjure Wares and Assemble Filesets.
			- Spawn containers.  Run things.
			- Teardown filesystem, save new Wares.
			- Tell you about it.

	Things that should already have been done:

		- One metric ton of args parsing: the CLI (or whatever other caller)
		   should have finished this by now.  The formula is *done*.
		- The selection of executors.  We just want to shell out to one.
		- The selection of transmats.  We just want to shell out to some.
*/
func (cfg Runner) Run() error {
	return nil
}

// this Run method accepts a context, and that's the one to kill the channel that's reporting log events.
// the Runner setup itself *also* needs a context, and that's the one that actual owns the real operations.
// ... review that, i'm not sure how sane those last two lines are

/*
	bad ideas:

	```
	type Constraints struct {
		Executor struct {
			Must    string
			MustNot []string
		}
	}
	```

	not useful to make part of remote api, because this, and the
	similar transmat placer constraints you might imagine, is supposed
	to be entirely *operational*.  your ops team decides copyplacer is
	unacceptable performance and would rather crash.  or that chroot
	is unacceptable containment (it is) and would rather crash.  etc.

	Think about this in terms of r2k8s.
	The setup that r2k8s launcher requires is A) substantial
	and B) completely unique to r2k8s.

	Retry:

		- Layer API-gen: has `Run(ctx, frm, disco, <-watch) (runrecord, err)` method.
		  - The amount of stuff you had to factory to *get* that is not my problem.
		- Layer real: yep, we fly that `Run()` flag.
		  - We need the following setup:
		    - workspace paths
		    - constraints on executors
		    - transmat plugins
		    - constraints on fileset assemblers
		  - When running from our own CLI, we tend to pick up the 'disco' helper info from our own cwd and environment.
		    - Not yet seen: "workspace" config, which *is* in fact diff than hitch: it's *yours* (much like .git/config urls vs the committed ones).
		    - So in this one case: the entry point (the CLI) has influence over *both* setup and the `Run()` args.  This is not the norm; don't get distracted by it.
		- Proxied by r2k8s: yep, we fly that `Run()` flag.
		  - We need the following setup:
		    - a k8s config file, dial info, and auth tokens.
		    - maybe some mapping info on things we're allowed to automagically hoistmirror?
		  - You actually lose the ability to specify some things:
		    - basically everything.  it's the cluster's decision.
		- A mock version: yep, we fly that `Run()` flag.
		  - We need the following setup:
		    - *nothing* because we're going to only follow certain very silly kinds of instructions.
*/
