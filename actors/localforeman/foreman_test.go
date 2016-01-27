package localforeman

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/model/cassandra/impl/mem"
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
)

func Test(t *testing.T) {
	Convey("Given a small knowledge base", t, func(c C) {
		kb := cassandra_mem.New()
		// publish artifact "apollo" -- default track only, single release
		kb.PublishCatalog(&catalog.Book{
			catalog.ID("apollo"),
			map[string][]catalog.SKU{"": []catalog.SKU{
				{"tar", "a1"},
			}},
		})
		// publish artifact "balogna" -- default track only, two releases
		kb.PublishCatalog(&catalog.Book{
			catalog.ID("balogna"),
			map[string][]catalog.SKU{"": []catalog.SKU{
				{"tar", "b1"},
				{"tar", "b2"},
			}},
		})
		// publish a commission -- something to interact with
		kb.PublishCommission(&formula.Commission{
			ID: formula.CommissionID("yis"),
			Formula: def.Formula{ // this inclusion is clunky, wtb refactor
				Inputs: def.InputGroup{
					"apollo": &def.Input{},
				},
			},
		})

		Convey("Formulas are emitted for all plans using latest editions of catalogs", func() {
			mgr := &Foreman{
				cassy: kb,
			}
			mgr.register()
			pumpn(mgr, 2)

			So(mgr.currentPlans.queue, ShouldHaveLength, 1)
		})
	})
}

func pumpn(mgr *Foreman, n int) {
	for i := 0; i < n; i++ {
		mgr.pump()
	}
}
