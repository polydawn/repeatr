package cassandra

import (
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
	"sync"
)

/*
	Sees everything; powerless to change it.

	This knowledgebase keeps records of all runs, all formulas, and all
	catalogs.  It accepts updates to them, dispatches messages on changes,
	and proxies any reads.  See the `actors` package for systems that
	alter the knowledgebase.
*/
type Base struct {
	mutex    sync.Mutex
	plans    map[formula.PlanID]*formula.Plan
	catalogs map[catalog.ID]*catalog.Book
	formulas map[formula.Stage2ID]*formula.Stage2
	results  map[formula.Stage3ID]*formula.Stage3

	catalogObservers []chan<- catalog.ID
}

func New() *Base {
	return &Base{
		plans:    make(map[formula.PlanID]*formula.Plan),
		catalogs: make(map[catalog.ID]*catalog.Book),
		formulas: make(map[formula.Stage2ID]*formula.Stage2),
		results:  make(map[formula.Stage3ID]*formula.Stage3),
	}
}

/*
	List all current catalog IDs.

	In order to have an ongoing concurrency-safe interaction with the set
	of known catalogs, subscribe to updates first, then ask for this list,
	then maintain merging those sets.
*/
func (kb *Base) ListCatalogs() []catalog.ID {
	// This might be advised to return an iterator later.
	kb.mutex.Lock()
	defer kb.mutex.Unlock()
	ret := make([]catalog.ID, 0, len(kb.catalogs))
	for k := range kb.catalogs {
		ret = append(ret, k)
	}
	return ret
}

func (kb *Base) PublishCatalog(book *catalog.Book) {
	kb.mutex.Lock()
	kb.catalogs[book.ID()] = book
	var observers []chan<- catalog.ID
	copy(kb.catalogObservers, observers)
	kb.mutex.Unlock()
	for _, obvs := range observers {
		obvs <- book.ID()
	}
}

func (kb *Base) Catalog(id catalog.ID) *catalog.Book {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()
	return kb.catalogs[id]
}

func (kb *Base) SelectPlansByInputCatalog(catIDs ...catalog.ID) []*formula.Plan {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()
	markedSet := make([]*formula.Plan, 0)
	for _, plan := range kb.plans {
		for iname, _ := range plan.Inputs { // INDEXABLE
			for _, catID := range catIDs {
				if iname == string(catID) {
					markedSet = append(markedSet, plan)
				}
			}
		}
	}
	return markedSet
}
