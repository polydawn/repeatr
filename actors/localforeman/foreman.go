package localforeman

import (
	"os"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/model/cassandra"
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
)

type Foreman struct {
	// configuration

	cassy    *cassandra.Base
	executor executor.Executor

	// work state

	chNewCatalog <-chan catalog.ID
	chOldCatalog <-chan catalog.ID
	currentPlans currentPlans
}

func (man *Foreman) work() {
	man.register()
	for {
		man.pump()
		man.evoke()
	}
}

// runs once upon start, rigging up our event feeds
func (man *Foreman) register() {
	// Register for catalog changes.
	chNewCatalog := make(chan catalog.ID, 100)
	man.cassy.ObserveCatalogs(chNewCatalog)
	man.chNewCatalog = chNewCatalog

	// Grab all current catalogs.  Give em one due consideration.
	// Dump em into a channel so we can select freely between these and fresh updates.
	// If an update careens in for one of these, and we react to that first, that's
	//  completely AOK: it'll end up nilled out when we reduce to stage2 formulas;
	//  the whole thing is an "at least once" situation.
	// We operate on CatalogIDs here instead of the full struct for two reasons:
	// - it's cheaper, if you didn't already have the whole thing loaded
	// - it means when you get the memo, you go get the latest -- and this
	//  absolves a race between old and updated catalogs in select.
	oldCats := man.cassy.ListCatalogs()
	oldCatalogChan := make(chan catalog.ID, len(oldCats))
	for _, cat := range oldCats {
		oldCatalogChan <- cat
	}
}

// runs in a loop, accepting events, generating new formulas, and adding them to currentPlans.
func (man *Foreman) pump() {
	// Select a new and interesting catalog.
	var catID catalog.ID
	select {
	case catID = <-man.chNewCatalog: // Voom
	case catID = <-man.chOldCatalog: // Voom
	}

	// 'Mark' phase: See what we can do with it.
	markedSet := man.cassy.SelectCommissionsByInputCatalog(catID)

	// 'Fill' phase.
	formulas := make([]*formula.Stage2, 0)
	for _, plan := range markedSet {
		formula := (*formula.Stage2)(plan) // FIXME need clone func and sane mem owner defn
		for iname, input := range formula.Inputs {
			cellID := catalog.ID(iname)                                  // this may not always be true / this is the same type haze around pre-pin inputs showing again
			input.Hash = string(man.cassy.Catalog(cellID).Latest().Hash) // this string cast is because def is currently Wrong
		}
		formulas = append(formulas, formula)
	}

	// 'Seenit' filter.
	// TODO
	// Compute Stage2 identifiers and index by that.  If it's been seen before, forget it.

	// Commit phase: push the stage2 formula back to the knowledge base.
	// TODO

	// Planning phase: update our internal concept of what's up next.
}

/*
	An atom capturing the foreman's current best idea of what formulas
	it wants to evaluate next.

	This is stateful because the foreman acknowledges info and produces
	new plans at a different pace than it can execute their evaluation,
	and it may also decide to cancel some plans in response to new info.
	(Also, it's a checkpoint for use in testing.)
*/
type currentPlans struct {
	// flat list of what formulas we want to run next, in order.
	queue []*formula.Stage2

	// map from cmid to queue index (so we can delete/replace things if they're now out of date).
	commissionIndex map[formula.CommissionID]int
}

func (man *Foreman) evoke() {
	// Run.
	for _, formula := range man.currentPlans.queue {
		job := man.executor.Start(def.Formula(*formula), def.JobID(guid.New()), nil, os.Stderr)
		job.Wait()
	}
	// TODO all sorts of other housekeeping on the queue

	// Commit phase: push the stage3 formulas back to storage.
	// TODO
	// If someone wants to react to these new run records by publishing
	//  a new edition of a catalog, they can do that by asking
	//   cassy to observe new run records like this one as they come in.
}
