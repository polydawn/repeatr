package catalog

type Watchable interface {
	/*
		Subscribe to updates to a catalog.  A new catalog will be sent to
		the channel every time there's new edition becomes available.

		This subscription only makes it about as far as your local post
		office -- you'll get notifications when a new edition makes it
		*there*; it doesn't guarantee the publishes are actively
		pushing notifications that far, or that the office is continuously
		checking upstream.  For example, a catalog based on local dirs
		might well scan continuously, but a git catalog probably doesn't
		poll the remote ever 100ns (but it might get webhooks that
		trigger a looksee, too!).
	*/
	Subscribe(chan<- Catalog)
}

// LATER:
//   We may also end up with a system of notifiers for newly recorded stage3
// execution completions, as well as task leases.  The first gen work planner
// is going to make simplifying assumptions about a single executor (it's local)
// and the main source of complicated interactions with the work heap is going
// to be cancellation if a new catalog edition comes out that makes current
// planned work less interesting -- but other events may be part of the mix in
// the future: some work items will be queued multiple times (i.e. check reps),
// some tasks may need to be farmed to different label groups (controlling for
// hardware), sometimes the goal is broad coverage and not just latest stuff,
// etc.  So, in short, planning details should not leak directly into
// anything about catalogs or run record and formula database features;
// those things need to be attached to and scoped within work planners.
