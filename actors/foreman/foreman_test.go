package foreman

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
	// okay, more releases now
	cat_apollo2 = &catalog.Book{
		catalog.ID("apollo"),
		map[string][]catalog.SKU{"": []catalog.SKU{
			{"tar", "a1"},
			{"tar", "a2"},
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
	// commission consuming nothing relevant
	cmsh_narp = &formula.Commission{
		ID: formula.CommissionID("narp"),
		Formula: def.Formula{ // this inclusion is clunky, wtb refactor
			Inputs: def.InputGroup{
				"whatever": &def.Input{},
			},
		},
	}

	// commission consuming apollo
	cmsh_yis = &formula.Commission{
		ID: formula.CommissionID("yis"),
		Formula: def.Formula{ // this inclusion is clunky, wtb refactor
			Inputs: def.InputGroup{
				"apollo": &def.Input{},
			},
		},
	}

	// comission consuming both apollo and balogna
	cmsh_whoosh = &formula.Commission{
		ID: formula.CommissionID("woosh"),
		Formula: def.Formula{ // this inclusion is clunky, wtb refactor
			Inputs: def.InputGroup{
				"apollo":  &def.Input{},
				"balogna": &def.Input{},
			},
		},
	}
)

func TestBasicPlanning(t *testing.T) {
	Convey("Foreman should generate plans from commissions in response to catalog delivery", t, func(c C) {
		Convey("Given a knowledge base with just some catalogs", func() {
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

		Convey("Given a knowledge base with some catalogs and somes commissions", func() {
			kb := cassandra_mem.New()
			kb.PublishCatalog(cat_apollo1)
			kb.PublishCatalog(cat_balogna2)
			kb.PublishCommission(cmsh_narp)
			kb.PublishCommission(cmsh_yis)

			mgr := &Foreman{
				cassy: kb,
			}
			mgr.register()

			Convey("Formulas are emitted for all plans using latest editions of catalogs", func() {
				pumpn(mgr, 2)

				// this is actually testing multiple things: related comissions are triggered,
				//  and also unrelated *aren't*.
				plans := mgr.currentPlans
				So(plans.queue, ShouldHaveLength, 1)
				So(plans.queue[0].formula.Inputs["apollo"], ShouldNotBeNil)
				So(plans.queue[0].formula.Inputs["apollo"].Hash, ShouldEqual, "a1")
			})

			Convey("After crashing more catalogs in concurrently", func() {
				kb.PublishCatalog(cat_apollo2)
				So(kb.ListCatalogs(), ShouldHaveLength, 2)

				Convey("Formulas are emitted for all plans using latest editions of catalogs", func() {
					pumpn(mgr, 3)

					// We should still only have one formula...
					// The semantics of `select` mean there may or may not have been two *generated*,
					// but since they share the same commission, one should be dropped.
					// There's also no danger of the "newer" one being dropped, since catalog notifications are by ID, not content.
					plans := mgr.currentPlans
					So(plans.queue, ShouldHaveLength, 1)
					So(plans.queue[0].formula.Inputs["apollo"], ShouldNotBeNil)
					So(plans.queue[0].formula.Inputs["apollo"].Hash, ShouldEqual, "a2")
				})
			})
		})

		Convey("Given a knowledge base with some catalogs and several relevant commissions", func() {
			kb := cassandra_mem.New()
			kb.PublishCatalog(cat_apollo1)
			kb.PublishCatalog(cat_balogna2)
			kb.PublishCommission(cmsh_narp)
			kb.PublishCommission(cmsh_yis)
			kb.PublishCommission(cmsh_whoosh)

			mgr := &Foreman{
				cassy: kb,
			}
			mgr.register()

			Convey("Formulas are emitted for all plans using latest editions of catalogs", func() {
				pumpn(mgr, 2)

				plans := mgr.currentPlans
				So(plans.queue, ShouldHaveLength, 2)
				So(plans.queue[0].formula.Inputs["apollo"], ShouldNotBeNil)
				So(plans.queue[0].formula.Inputs["apollo"].Hash, ShouldEqual, "a1")
				So(plans.queue[1].formula.Inputs["apollo"], ShouldNotBeNil)
				So(plans.queue[1].formula.Inputs["apollo"].Hash, ShouldEqual, "a1")
				// look at the current commission records; they can be in either order
				So(plans.commissionIndex, ShouldHaveLength, 2)
				idx_yis := plans.commissionIndex[cmsh_yis.ID]
				idx_woosh := plans.commissionIndex[cmsh_whoosh.ID]
				So(idx_woosh+idx_yis, ShouldEqual, 1)

				Convey("After crashing more catalogs in concurrently", func() {
					kb.PublishCatalog(cat_apollo2)
					So(kb.ListCatalogs(), ShouldHaveLength, 2)

					Convey("Formulas from the same commission are replaced", func() {
						pumpn(mgr, 1)

						plans := mgr.currentPlans
						So(plans.queue, ShouldHaveLength, 2)
						So(plans.queue[0].formula.Inputs["apollo"], ShouldNotBeNil)
						So(plans.queue[0].formula.Inputs["apollo"].Hash, ShouldEqual, "a2")
						So(plans.queue[1].formula.Inputs["apollo"], ShouldNotBeNil)
						So(plans.queue[1].formula.Inputs["apollo"].Hash, ShouldEqual, "a2")
						// commission records can still be in either order, just has to be same
						So(plans.commissionIndex, ShouldHaveLength, 2)
						So(plans.commissionIndex[cmsh_yis.ID], ShouldEqual, idx_yis)
						So(plans.commissionIndex[cmsh_whoosh.ID], ShouldEqual, idx_woosh)
					})
				})
			})
		})
	})
}

func pumpn(mgr *Foreman, n int) {
	for i := 0; i < n; i++ {
		mgr.pump()
	}
}
