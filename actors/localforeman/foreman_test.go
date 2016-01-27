package localforeman

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/model/cassandra/impl/mem"
	"polydawn.net/repeatr/model/catalog"
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

		Convey("Formulas are emitted for all plans using latest editions of catalogs", func() {
			mgr := &Foreman{
				cassy: kb,
			}
			mgr.register()
			mgr.pump()
			mgr.pump()

			// There *are* no plans if there's just catalogs and no formulas!
			So(mgr.currentPlans.queue, ShouldHaveLength, 0)
		})
	})
}
