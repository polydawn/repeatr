package foreman

import (
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rsrch/model/formula"
)

type plan struct {
	formula        *formula.Stage2
	commissionedBy formula.CommissionID
	leasedAs       leaseToken
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
	// not a priority queue.  just plain straight FIFO.
	queue []*plan

	// map indexing plans by lease
	leasesIndex map[leaseToken]*plan

	// map indexing plans by cmid (so we can delete/replace things if they're now out of date).
	// if something has been leased, it's outta here (you can't replace an in progress task!).
	commissionIndex map[formula.CommissionID]*plan
}

type leaseToken string

func NewPlans() *plans {
	return &plans{
		queue:           make([]*plan, 0, 10),
		leasesIndex:     make(map[leaseToken]*plan),
		commissionIndex: make(map[formula.CommissionID]*plan),
	}
}

func (ps *plans) Push(p *plan) {
	// look for any never-leased plans from the same commission; if
	//  any found, kick them out of the queue and take their place.
	if pSameCmid, ok := ps.commissionIndex[p.commissionedBy]; ok {
		for i, qp := range ps.queue {
			if qp == pSameCmid {
				ps.queue[i] = p
				ps.commissionIndex[p.commissionedBy] = p
				return
			}
		}
	}

	// otherwise, we'll get in line at the back.
	ps.queue = append(ps.queue, p)
	ps.commissionIndex[p.commissionedBy] = p
}

/*
	Request the next available task, and take out a lease on it;
	returns nil if no available tasks.
	`LeaseNext` should be immediatedly followed with a
	`defer Unlease(ltok)`, and a `Finish` on success.
*/
func (ps *plans) LeaseNext() (*plan, leaseToken) {
	// Pick one out.
	i := ps.nextLeasable()
	if i < 0 {
		return nil, ""
	}
	p := ps.queue[i]
	// Assign a lease token.
	p.leasedAs = leaseToken(guid.New())
	ps.leasesIndex[p.leasedAs] = p
	// Drop it from the commissionIndex; it's no longer evictable.
	delete(ps.commissionIndex, p.commissionedBy)
	// Hand it out.
	// (We return the token separately because that one's immutable;
	//  the one on the plan is our queue's interally up-to-date concept.)
	return p, p.leasedAs
}

/*
	Return a lease, while considering the task incomplete.  `LeaseNext`
	calls may then get this task again.  Original order is retained;
	`Unlease`ing a really old task almost certainly means it will be drawn next.

	Calling `Unlease` repeatedly is fine; it's idempotent (seealso `Finish`).
*/
func (ps *plans) Unlease(ltok leaseToken) {
	p, ok := ps.leasesIndex[ltok]
	if !ok {
		return
	}
	p.leasedAs = ""
	delete(ps.leasesIndex, ltok)
}

/*
	Call the thing done, and remove it from the queue.

	If `Unlease` is called after this, it'll be a noop.
	If `Unlease` is called *before* this, then *this* is a noop.
	(The latter isn't really ideal, but it's a neccessary boundary so we
	aren't compelled to keep lease tokens around eternally.)
*/
func (ps *plans) Finish(lease leaseToken) {
	p, ok := ps.leasesIndex[lease]
	if !ok {
		return
	}
	ps.remove(p)
}

func (ps *plans) remove(p *plan) {
	// drop from queue
	for i, qp := range ps.queue {
		if qp == p {
			ps.queue = append(ps.queue[:i], ps.queue[i+1:]...)
			break
		}
	}
	// drop from leases (easy, and possibly noop)
	delete(ps.leasesIndex, p.leasedAs)
	// drop from commission index... only if it was pointing at us
	// (this one is complicated because another plan from the same commission
	//   may have been enqueued while this one was leased.)
	latestPlanForCommission, ok := ps.commissionIndex[p.commissionedBy]
	if ok && latestPlanForCommission == p {
		delete(ps.commissionIndex, p.commissionedBy)
	}
}

func (ps *plans) nextLeasable() int {
	for i, p := range ps.queue {
		if p.leasedAs == "" {
			return i
		}
	}
	return -1
}
