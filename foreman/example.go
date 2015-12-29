package foreman

import (
	"polydawn.net/repeatr/catalog"
	"polydawn.net/repeatr/def/derive"
)

/*
	A Foreman implementation implicitly has to keep some sort of loosely
	constructed, possibly-not-fully-connected graph internally.
*/
type Foreman interface {
	AddCatalogWatch(catalog.Watchable)
}

/*
	Sees everything; powerless to change it.

	This interface is likely to end up backed by a SQL DB.
	Note to future self: try to avoid the API clusterfuck that usually
	ensues from "interface that bundles a bunch of queries" (also seen
	in jgit and git2go alike).
*/
type Cassandra interface {
	SelectAllFormulas() <-chan derive.Stage2
	SelectAllResults(derive.Stage2) <-chan derive.Stage3
	SelectAllRunRecords(derive.Stage3) <-chan RunRecord // TODO:ITR
}

type RunRecord struct {
	// placeholder.
}
