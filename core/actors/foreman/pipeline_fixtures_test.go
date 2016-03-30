package foreman

import (
	"polydawn.net/repeatr/core/model/cassandra"
	"polydawn.net/repeatr/core/model/formula"
	"polydawn.net/repeatr/def"
)

/*
	Load a whole series of commissions which form
	roughly the following tree of [catalogs] and <<commissions>>:

	  [A] ----- <<B>> ----> [B::x] ---- <<E>> ---> [E::x]
	    \
	     \___ <<D>> ----> [D::x]
	     /          \
	    /            \___> [D::y]
	  [C]

	This covers:
	  - A primitive chain (between 'B' and 'E')
	  - Fan in (at commission 'D')
	  - Fan out (also at commission 'D')
	It does not cover:
	  - A diamond.  We'll do that one challenge level 2.
	  - Non-conjectured steps.  They're... less interesting.

	All transitions are computed with the null executor
	in deterministic mode (no bad/interesting behaviors).
*/
func commissionChallengeSuiteOne(kb cassandra.Cassandra) {
	kb.PublishCommission(&formula.Commission{
		ID: formula.CommissionID("B"),
		Formula: def.Formula{
			Inputs: def.InputGroup{
				"A": &def.Input{},
			},
			Outputs: def.OutputGroup{
				"x": &def.Output{},
			},
		},
	})
	kb.PublishCommission(&formula.Commission{
		ID: formula.CommissionID("D"),
		Formula: def.Formula{
			Inputs: def.InputGroup{
				"A": &def.Input{},
				"C": &def.Input{},
			},
			Outputs: def.OutputGroup{
				"x": &def.Output{},
				"y": &def.Output{},
			},
		},
	})
	kb.PublishCommission(&formula.Commission{
		ID: formula.CommissionID("E"),
		Formula: def.Formula{
			Inputs: def.InputGroup{
				"B::x": &def.Input{},
			},
			Outputs: def.OutputGroup{
				"x": &def.Output{},
			},
		},
	})
}
