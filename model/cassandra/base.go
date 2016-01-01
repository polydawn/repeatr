package cassandra

import (
	"polydawn.net/repeatr/model/catalog"
)

/*
	Sees everything; powerless to change it.

	This knowledgebase keeps records of all runs, all formulas, and all
	catalogs.  It accepts updates to them, dispatches messages on changes,
	and proxies any reads.  See the `actors` package for systems that
	alter the knowledgebase.
*/
type Base struct {
}

/*
	List all current catalog IDs.

	In order to have an ongoing concurrency-safe interaction with the set
	of known catalogs, subscribe to updates first,	then ask for this list,
	then maintain merging those sets.
*/
func (base *Base) ListCatalogs() []catalog.ID {
	// This might be advised to return an iterator later.

	return nil // TODO NYI
}

func (base Base) Catalog(id catalog.ID) *catalog.Book {
	return nil // TODO NYI
}
