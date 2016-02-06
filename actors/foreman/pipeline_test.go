package foreman

import (
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/executor/null"
	"polydawn.net/repeatr/model/cassandra/impl/mem"
	"polydawn.net/repeatr/model/catalog"
)

func TestPipeline(t *testing.T) {
	/*
	  [A] ----- <<B>> ----> [B::x] ---- <<E>> ---> [E::x]
	    \
	     \___ <<D>> ----> [D::x]
	     /          \
	    /            \___> [D::y]
	  [C]
	*/
	Convey("Given challenge suite one", t, func(c C) {
		kb := cassandra_mem.New()
		// load up the suite
		commissionChallengeSuiteOne(kb)
		// make a foreman including a full mock executor, and register it up
		mgr := &Foreman{
			cassy:    kb,
			executor: &null.Executor{null.Deterministic},
		}
		mgr.register()

		Convey("Delivering catalog 'A' triggers work", func() {
			kb.PublishCatalog(catalog.New(catalog.ID("A")).Release(
				"", catalog.SKU{Hash: "a1"},
			))
			pumpn(mgr, 1)

			// we should get B to run, and not D.
			So(mgr.currentPlans.queue, ShouldHaveLength, 1)
			mgr.evoke()
			So(mgr.currentPlans.queue, ShouldHaveLength, 0)

			// we should get B releases!
			allCatIDs := kb.ListCatalogs()
			So(allCatIDs, ShouldHaveLength, 2)
			sort.Sort(catalog.IDs(allCatIDs))
			So(allCatIDs[0], ShouldEqual, "A")
			So(allCatIDs[1], ShouldEqual, "B::x")
			So(kb.Catalog(catalog.ID("B::x")).Tracks[""], ShouldHaveLength, 1)
		})
	})
}
