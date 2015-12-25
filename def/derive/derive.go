package derive

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
)

type Potential struct {
	Plans          map[PlanID]Plan
	LastResolution map[PlanID]Stage3
	Cells          map[CellID]Cell
}

type PlanID string // good luck making this not hopelessly imperial.  probably be local-only.
// actually no... these can be CA too.  they just aren't so required to collide because Plans refer to cell.Name.
// not sure if that matters: that's *possible*, but still not exactly necessarily *useful* because *plans change*
//  and often we want naming continuity across that.
//  But it is worth considering the possibility that the naming contiguity really only matters on the cells, and nobody cares on the plans except as that they're referenced by the cells.
//   As an example of the practical impact of that, if a plan has been replaced by a newer version, we have *zero* desire to keep feeding other cell changes into the old one and doing work based on it.
//    You could almost encapsulate plans themselves in a cell and get surprisingly reasonable semantics out of that arrangement.  If terrible documentation from the over-abstraction.

type Plan def.Formula // placeholder, will have different types for inputs obviously

type Stage2 def.Formula

type Stage3 def.Formula

type CellID string

type Cell interface {
	Name() CellID // here is the true imperial strike

	Latest() integrity.CommitID

	// so, the fun thing about this interface is: you probably
	//  want to have a way to let it promote itself as changed.
	//  Think e.g. 'watch' on the filesystem.
	//  Just generally, we don't want to poll on O(n) of these;
	//   most interactions i can think of involve the user triggering
	//    a specific update fetch (or a webhook, or whatever).
	// Maybe put that in a separate interface.
	//  Lol.  Updater updater?  Yeek.
	//   Maybe it's a good thing that FRP chat made me use a regular
	//    noun here instead of a gerund: this is a state.
}

// Imagine this being triggered by `change := <-chan Cell`
func (pot *Potential) Resolve(change Cell) { // ChartMap?
	// 'Mark' phase.
	// todo: split this; if we get many changes, this should batch.
	// note: we don't care if a change is a revert.  that'll just settle out later in the 'reruns' filter.
	markedSet := make(map[PlanID]struct{})
	for id, plan := range pot.Plans {
		for iname, _ := range plan.Inputs { // INDEXABLE
			if iname == string(change.Name()) {
				markedSet[id] = struct{}{}
			}
		}
	}

	// 'Fill' phase.
	formulas := make([]Stage2, 0)
	for id, _ := range markedSet {
		plan := pot.Plans[id]
		formula := Stage2(plan) // FIXME need clone func and sane mem owner defn
		for iname, input := range formula.Inputs {
			cellID := CellID(iname)                         // this may not always be true / this is the same type haze around pre-pin inputs showing again
			input.Hash = string(pot.Cells[cellID].Latest()) // this string cast is because def is currently Wrong
		}
		formulas = append(formulas, formula)
	}

	// 'Seenit' filter.
	// TODO
	// This *could* just use the LastResolution stuff but... why?
	// We can compute Stage2 identifiers and index by that and it's
	//  both *easier* and *more correct* than something that's scoped to PlanID.

	// Run.
	// TODO this is the easy part: shell out to the rest of repeatr.
	// TODO this should probably be able to emit acceptance-test formulas.

	// Commit.
	// ... This is the least-spec'd part right now.
	// TODO consider the possibility that there will be a big time gap between run and commit.
	//  For example, we might want to farm things to many machines, check
	//   repeatability, and only upon gathering resuls then commit... which is a much bigger
	//    op than even acceptance formulas.
	//  But this also clearly does not need to be in the first prototype.
}

// You could imagine trying to implement this "Mark" phase as a rolling
//  wave over a database where time proceeds as a list of cell change notifications.
//  Not clear how useful this idea would be.
//  Practically speaking, we might want the opposite in a sufficiently large
//   build farm: have a heap of queued tasks where we can readily invalidate
//    a task if it's been supplanted by an even more up-to-date plan resolution.
