package foreman

import (
	"polydawn.net/repeatr/model/formula"
)

type plan struct {
	formula        *formula.Stage2
	commissionedBy formula.CommissionID
}

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
	// flat list of what plans we want to run next, in order.  0->next.
	queue []*plan

	// map from cmid to queue index (so we can delete/replace things if they're now out of date).
	commissionIndex map[formula.CommissionID]int
}

func (ps *plans) push(p *plan) {
	if i, ok := ps.commissionIndex[p.commissionedBy]; ok {
		ps.queue[i] = p
	} else {
		i = len(ps.queue)
		ps.queue = append(ps.queue, p)
		ps.commissionIndex[p.commissionedBy] = i
	}
}

func (ps *plans) poll() *plan {
	l := len(ps.queue)
	if l == 0 {
		return nil
	}
	v := ps.queue[0]
	ps.queue = ps.queue[1:]
	return v
}
