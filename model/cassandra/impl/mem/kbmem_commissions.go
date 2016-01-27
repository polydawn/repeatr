package cassandra_mem

import (
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
)

func (kb *Base) SelectCommissionsByInputCatalog(catIDs ...catalog.ID) []*formula.Commission {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()
	markedSet := make([]*formula.Commission, 0)
	for _, plan := range kb.commissions {
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
