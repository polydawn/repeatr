package localforeman

import (
	"os"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/guid"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/model/cassandra"
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
)

type Foreman struct {
	cassy *cassandra.Base
	executor executor.Executor
}

func (man *Foreman) work() {
	// Register for catalog changes.
	catalogChan := make(chan catalog.ID, 100)
	man.cassy.ObserveCatalogs(catalogChan)

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

	for {
		// Select a new and interesting catalog.
		var catID catalog.ID
		select {
		case catID = <-catalogChan: // Voom
		case catID = <-oldCatalogChan: // Voom
		}
		cat := man.cassy.Catalog(catID)

		// 'Mark' phase: See what we can do with it.
		plans := man.getPlanList()
		markedSet := make([]*formula.Plan, 0)
		for _, plan := range plans {
			for iname, _ := range plan.Inputs { // INDEXABLE
				if iname == string(cat.ID()) {
					markedSet = append(markedSet, plan)
				}
			}
		}

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
		// This *could* just use the LastResolution stuff but... why?
		// We can compute Stage2 identifiers and index by that and it's
		//  both *easier* and *more correct* than something that's scoped to PlanID.

		// Run.
		for _, formula := range formulas {
			job := man.executor.Start(def.Formula(*formula), def.JobID(guid.New()), nil, os.Stderr)
			job.Wait()
		}

		// Commit.
		// TODO just push the stage3 formulas back to storage.
		// If someone wants to react to these new run records by publishing
		//  a new edition of a catalog, they can do that by asking
		//   cassy to observe new run records like this one as they come in.
	}
}

func (man *Foreman) getPlanList() []*formula.Plan {
	return nil // TODO NYI cassy should have this
	// can probably pretend this is immutable for first pass, but later
	//  will need the same updating strategy as catalogs, plus removal.
}
