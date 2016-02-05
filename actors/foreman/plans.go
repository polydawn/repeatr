package foreman

import (
	"polydawn.net/repeatr/model/formula"
)

/*
	An atom capturing the foreman's current best idea of what formulas
	it wants to evaluate next.

	This is stateful because the foreman acknowledges info and produces
	new plans at a different pace than it can execute their evaluation,
	and it may also decide to cancel some plans in response to new info.
	(Also, it's a checkpoint for use in testing.)

	Not yet particularly well covered:
	  - "seenit" behavior on stage2 formulas.  (You might be explicitly
	    requesting a re-execution; this level of the api doesn't know.)
*/
type plans struct {
	// flat list of what formulas we want to run next, in order.
	queue []*formula.Stage2

	// map from cmid to queue index (so we can delete/replace things if they're now out of date).
	commissionIndex map[formula.CommissionID]int
}

func (p *plans) push(f *formula.Stage2, reason formula.CommissionID) {
	if i, ok := p.commissionIndex[reason]; ok {
		p.queue[i] = f
	} else {
		i = len(p.queue)
		p.queue = append(p.queue, f)
		p.commissionIndex[reason] = i
	}
}

func (p *plans) poll() *formula.Stage2 {
	l := len(p.queue)
	if l == 0 {
		return nil
	}
	v := p.queue[0]
	p.queue = p.queue[1:]
	return v
}
