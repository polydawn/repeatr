package foreman

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

	cassy    cassandra.Cassandra
	executor executor.Executor

	// work state

	chNewCatalog <-chan catalog.ID
	chOldCatalog <-chan catalog.ID
	currentPlans *plans
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
	man.chOldCatalog = oldCatalogChan

	// other misc init (can't be arsed to seperate since it's also an "exactly once, at start" thing)
	man.currentPlans = NewPlans()
}

/*
	`Pump` accepts events, generates new formulas, and adds plans to run
	them to currentPlans.  Pumping should be run in a loop.  Pumping will
	block indefinitely if there are no new events from the knowledgebase.

	Formulas produced by pumping are not immediately committed back to the
	knowledge base; since the knowledgebase may GC any references not
	justified by a release catalog, it only makes sense to commit the
	formulas after they've been executed.
*/
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
	reasons := make(map[formula.CommissionID]*formula.Stage2)
	for _, commish := range markedSet {
		formula := (*formula.Stage2)(commish.Formula.Clone())
		for iname, input := range formula.Inputs {
			cellID := catalog.ID(iname)                                  // this may not always be true / this is the same type haze around pre-pin inputs showing again
			input.Hash = string(man.cassy.Catalog(cellID).Latest().Hash) // this string cast is because def is currently Wrong
		}
		formulas = append(formulas, formula)
		reasons[commish.ID] = formula
	}

	// Planning phase: update our internal concept of what's up next.
	for reason, formula := range reasons {
		man.currentPlans.Push(&plan{
			formula:        formula,
			commissionedBy: reason,
		})
	}
}

func (man *Foreman) evoke() {
	// Request a task.
	p, leaseToken := man.currentPlans.LeaseNext()
	if leaseToken == "" {
		return
	}
	// Automatically unlease it if something goes off the rails.
	//  (If we signal success, unlease is no-op'd.)
	defer man.currentPlans.Unlease(leaseToken)

	// Launch
	job := man.executor.Start(def.Formula(*p.formula), def.JobID(guid.New()), nil, os.Stderr)
	jobResult := job.Wait()
	man.currentPlans.Finish(leaseToken)
	// Assemble results // todo everything about jobresult is a mangle, plz refactor
	result := (*formula.Stage3)(def.Formula(*p.formula).Clone())
	result.Outputs = jobResult.Outputs

	// Any releases?
	newEditions := makeReleases(man.cassy, p, result)
	// TODO wrap up on committing them to the kb
	_ = newEditions

	// Commit phase: push the stage3 formulas back to storage.
	// TODO
	// We may also need to trigger release criteria *before* this, for
	//  the usual gc-strong-ref purposes.
}
