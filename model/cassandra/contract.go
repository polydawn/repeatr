package cassandra

import (
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
)

/*
	Sees everything; powerless to change it.

	This knowledgebase keeps records of all runs, all formulas, and all
	catalogs.  It accepts updates to them, dispatches messages on changes,
	and proxies any reads.  See the `actors` package for systems that
	alter the knowledgebase.
*/
type Cassandra interface {

	//////////////////////////////// Catalogs ////////////////////////////////

	Catalog(id catalog.ID) *catalog.Book

	/*
		List all current catalog IDs.

		In order to have an ongoing concurrency-safe interaction with the set
		of known catalogs, subscribe to updates first, then ask for this list,
		then maintain merging those sets.
	*/
	ListCatalogs() []catalog.ID

	/*
		Subscribe to updates to catalogs.  Every time there's a new edition of
		a catalog available, that catalog ID will be sent to the channel.

		This subscription only makes it about as far as your local post
		office -- you'll get notifications when a new edition makes it
		*there*; it doesn't guarantee the publishes are actively
		pushing notifications that far, or that the office is continuously
		checking upstream.  For example, a catalog based on local dirs
		might well scan continuously, but a git catalog probably doesn't
		poll the remote ever 100ns (but it might get webhooks that
		trigger a looksee, too!).

		Catalog ID (as opposed to the full book) are passed because you typically
		always want to interact with the latest version, and catalogs themselves
		do not contain links to their history or any ordering hints; so, you
		always need to respond to nudges by fetching then latest edition.
		(Specifically, this is meant to be a safeguard if you're listening to
		more than one buffered source of updates, and you need to be sure to
		behave correctly even if you heard about things in the wrong order.)
	*/
	ObserveCatalogs(ch chan<- catalog.ID)

	PublishCatalog(book *catalog.Book)

	//////////////////////////////// Commissions ////////////////////////////////

	ObserveCommissions(ch chan<- formula.CommissionID)

	PublishCommission(cmsh *formula.Commission)

	SelectCommissionsByInputCatalog(catIDs ...catalog.ID) []*formula.Commission
}
