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

		// This is a huge list list of steps, and will render horribly.
		// But I really do want to test a looong series of stateful transitions.
		type step struct {
			name string
			fn   func()
		}
		steps := []step{}
		steps = append(steps, step{
			"Deliver catalog 'A'; expect commission 'B' to fly",
			func() {
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
				sort.Sort(catalog.IDs(allCatIDs))
				So(allCatIDs, ShouldHaveLength, 2)
				So(allCatIDs[0], ShouldEqual, "A")
				So(allCatIDs[1], ShouldEqual, "B::x")
				So(kb.Catalog(catalog.ID("B::x")).Tracks[""], ShouldHaveLength, 1)
			},
		})
		steps = append(steps, step{
			"Pump; expect commission 'E' to fly using B's results",
			func() {
				// pump event from previous step
				pumpn(mgr, 1)

				// B doesn't need to run again, but E should be ready
				So(mgr.currentPlans.queue, ShouldHaveLength, 1)
				mgr.evoke()
				So(mgr.currentPlans.queue, ShouldHaveLength, 0)

				// we should get E releases!
				allCatIDs := kb.ListCatalogs()
				sort.Sort(catalog.IDs(allCatIDs))
				So(allCatIDs, ShouldHaveLength, 3)
				So(allCatIDs[0], ShouldEqual, "A")
				So(allCatIDs[1], ShouldEqual, "B::x")
				So(allCatIDs[2], ShouldEqual, "E::x")
				So(kb.Catalog(catalog.ID("B::x")).Tracks[""], ShouldHaveLength, 1)
				So(kb.Catalog(catalog.ID("E::x")).Tracks[""], ShouldHaveLength, 1)
			},
		})
		chain := func() {}
		for i := len(steps) - 1; i >= 0; i-- {
			step := steps[i]
			chain = func(chain func()) func() {
				return func() {
					Convey(step.name, func() {
						step.fn()
						chain()
					})
				}
			}(chain)
		}
		chain()
	})
}
