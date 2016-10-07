package foreman

import (
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/core/executor/impl/null"
	"go.polydawn.net/repeatr/rsrch/model/cassandra/impl/mem"
	"go.polydawn.net/repeatr/rsrch/model/catalog"
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

				// pump event 'E' (it's noise, shouldn't trigger anything)
				So(mgr.chNewCatalog, ShouldHaveLength, 1)
				pumpn(mgr, 1)
			},
		})
		steps = append(steps, step{
			"Deliver catalog 'C'; expect commission 'D' to fly",
			func() {
				kb.PublishCatalog(catalog.New(catalog.ID("C")).Release(
					"", catalog.SKU{Hash: "c1"},
				))
				pumpn(mgr, 1)

				// D has both required inputs now so it can run
				So(mgr.currentPlans.queue, ShouldHaveLength, 1)
				mgr.evoke()
				So(mgr.currentPlans.queue, ShouldHaveLength, 0)

				// we should get double D releases!
				allCatIDs := kb.ListCatalogs()
				sort.Sort(catalog.IDs(allCatIDs))
				So(allCatIDs, ShouldHaveLength, 6)
				So(allCatIDs[5], ShouldEqual, "E::x") // sorted, natch
				So(allCatIDs[2], ShouldEqual, "C")
				So(allCatIDs[3], ShouldEqual, "D::x")
				So(allCatIDs[4], ShouldEqual, "D::y")
				So(kb.Catalog(catalog.ID("D::x")).Tracks[""], ShouldHaveLength, 1)
				So(kb.Catalog(catalog.ID("D::y")).Tracks[""], ShouldHaveLength, 1)

				// pump both 'D::*' events (trailing noise)
				So(mgr.chNewCatalog, ShouldHaveLength, 2)
				pumpn(mgr, 2)
			},
		})
		steps = append(steps, step{
			"Deliver catalog 'A'->2; expect all hell to break loose, eeeeverybody gets a rebuild",
			func() {
				kb.PublishCatalog(
					kb.Catalog(catalog.ID("A")).
						Release("", catalog.SKU{Hash: "a2"}),
				)
				pumpn(mgr, 1)

				// both D and B consume this and neither have incomplete reqs
				// no idea what order they're in!  shouldn't matter!  evoke twice.
				So(mgr.currentPlans.queue, ShouldHaveLength, 2)
				mgr.evoke()
				mgr.evoke()
				// E still hasn't triggered because evoke doesn't pump again
				So(mgr.currentPlans.queue, ShouldHaveLength, 0)

				// D (x2) and B both expect new releases
				allCatIDs := kb.ListCatalogs()
				sort.Sort(catalog.IDs(allCatIDs))
				So(allCatIDs, ShouldHaveLength, 6)
				So(kb.Catalog(catalog.ID("B::x")).Tracks[""], ShouldHaveLength, 2)
				So(kb.Catalog(catalog.ID("D::x")).Tracks[""], ShouldHaveLength, 2)
				So(kb.Catalog(catalog.ID("D::y")).Tracks[""], ShouldHaveLength, 2)

				// trailin 'D::*' and a meaningful 'B::x' event
				So(mgr.chNewCatalog, ShouldHaveLength, 3)
				pumpn(mgr, 3)

				// knock out the E again
				So(mgr.currentPlans.queue, ShouldHaveLength, 1)
				mgr.evoke()
				So(mgr.currentPlans.queue, ShouldHaveLength, 0)
				So(kb.Catalog(catalog.ID("E::x")).Tracks[""], ShouldHaveLength, 2)
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
