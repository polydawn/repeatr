package cassandra

/*
	Sees everything; powerless to change it.

	This knowledgebase keeps records of all runs, all formulas, and all
	catalogs.  It accepts updates to them, dispatches messages on changes,
	and proxies any reads.  See the `actors` package for systems that
	alter the knowledgebase.
*/
type Base struct {
}
