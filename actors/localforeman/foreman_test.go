package localforeman

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/model/cassandra/impl/mem"
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
)

var (
	// artifact "apollo" -- default track only, single release
	cat_apollo1 = &catalog.Book{
		catalog.ID("apollo"),
		map[string][]catalog.SKU{"": []catalog.SKU{
			{"tar", "a1"},
		}},
	}

	// artifact "balogna" -- default track only, two releases
	cat_balogna2 = &catalog.Book{
		catalog.ID("balogna"),
		map[string][]catalog.SKU{"": []catalog.SKU{
			{"tar", "b1"},
			{"tar", "b2"},
		}},
	}
)

var (
	// commission consuming apollo
	cmsh_yis = &formula.Commission{
		ID: formula.CommissionID("yis"),
		Formula: def.Formula{ // this inclusion is clunky, wtb refactor
			Inputs: def.InputGroup{
				"apollo": &def.Input{},
			},
		},
	}
)

func Test(t *testing.T) {
	Convey("Given a knowledge base with just some catalogs", t, func(c C) {
		kb := cassandra_mem.New()
		kb.PublishCatalog(cat_apollo1)
		kb.PublishCatalog(cat_balogna2)

		Convey("Foreman plans no formulas because there are no commissions", func() {
			mgr := &Foreman{
				cassy: kb,
			}
			mgr.register()
			pumpn(mgr, 2)

			So(mgr.currentPlans.queue, ShouldHaveLength, 0)
		})
	})

	Convey("Given a knowledge base with some catalogs and a commission", t, func(c C) {
		kb := cassandra_mem.New()
		kb.PublishCatalog(cat_apollo1)
		kb.PublishCatalog(cat_balogna2)
		kb.PublishCommission(cmsh_yis)

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
