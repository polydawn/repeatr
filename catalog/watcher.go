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
