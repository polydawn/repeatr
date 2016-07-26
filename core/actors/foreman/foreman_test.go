package foreman

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/model/cassandra/impl/mem"
	"go.polydawn.net/repeatr/core/model/catalog"
	"go.polydawn.net/repeatr/core/model/formula"
)

var (
	// artifact "apollo" -- default track only, single release
	cat_apollo1 = catalog.New(catalog.ID("apollo")).
			Release("", catalog.SKU{"tar", "a1"})
	// okay, more releases now
	cat_apollo2 = cat_apollo1.
			Release("", catalog.SKU{"tar", "a2"})

	// artifact "balogna" -- default track only, two releases
	cat_balogna2 = catalog.New(catalog.ID("balogna")).
			Release("", catalog.SKU{"tar", "b1"}).
			Release("", catalog.SKU{"tar", "b2"})

	// artifact "falsetto" -- it's a trap!  no actual releases!
	cat_falsetto0 = catalog.New(catalog.ID("falsetto"))
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

	// comission consuming apollo and falsetto (the trick)
	cmsh_xilence = &formula.Commission{
		ID: formula.CommissionID("xilence"),
		Formula: def.Formula{ // this inclusion is clunky, wtb refactor
			Inputs: def.InputGroup{
				"apollo":   &def.Input{},
				"falsetto": &def.Input{},
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

			Convey("Formulas are emitted for all commissions using latest editions of catalogs", func() {
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

				Convey("Formulas are emitted for each commission using the updated catalogs", func() {
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

			Convey("Formulas are emitted for all commissions using latest editions of catalogs", func() {
				pumpn(mgr, 2)

				plans := mgr.currentPlans
				So(plans.queue, ShouldHaveLength, 2)
				So(plans.queue[0].formula.Inputs["apollo"], ShouldNotBeNil)
				So(plans.queue[0].formula.Inputs["apollo"].Hash, ShouldEqual, "a1")
				So(plans.queue[1].formula.Inputs["apollo"], ShouldNotBeNil)
				So(plans.queue[1].formula.Inputs["apollo"].Hash, ShouldEqual, "a1")
				So(plans.commissionIndex, ShouldHaveLength, 2)

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
						So(plans.commissionIndex, ShouldHaveLength, 2)
					})
				})

				Convey("Leasing some tasks should work", func() {
					plans := mgr.currentPlans
					p, ltok := plans.LeaseNext()
					So(p, ShouldNotBeNil)
					So(ltok, ShouldNotResemble, "")

					Convey("Queue should remain, but commissionIndex drop", func() {
						So(plans.queue, ShouldHaveLength, 2)
						So(plans.commissionIndex, ShouldHaveLength, 1)
						So(plans.leasesIndex, ShouldHaveLength, 1)
					})

					Convey("After crashing more catalogs in concurrently", func() {
						kb.PublishCatalog(cat_apollo2)
						So(kb.ListCatalogs(), ShouldHaveLength, 2)

						Convey("Already leased plans should not be replaced", func() {
							pumpn(mgr, 1)

							plans := mgr.currentPlans
							So(plans.queue, ShouldHaveLength, 3)
							So(plans.queue[0].formula.Inputs["apollo"], ShouldNotBeNil)
							So(plans.queue[0].formula.Inputs["apollo"].Hash, ShouldEqual, "a1")
							So(plans.queue[1].formula.Inputs["apollo"], ShouldNotBeNil)
							So(plans.queue[1].formula.Inputs["apollo"].Hash, ShouldEqual, "a2")
							So(plans.queue[1].formula.Inputs["apollo"], ShouldNotBeNil)
							So(plans.queue[1].formula.Inputs["apollo"].Hash, ShouldEqual, "a2")
							So(plans.commissionIndex, ShouldHaveLength, 2)
							So(plans.leasesIndex, ShouldHaveLength, 1)
						})
					})

					Convey("Finishing a plan should remove it", func() {
						plans.Finish(ltok)
						So(plans.queue, ShouldHaveLength, 1)
						So(plans.commissionIndex, ShouldHaveLength, 1)
						So(plans.leasesIndex, ShouldHaveLength, 0)

						Convey("Unleasing it afterward should no-op", func() {
							plans.Unlease(ltok)
							So(plans.queue, ShouldHaveLength, 1)
							So(plans.commissionIndex, ShouldHaveLength, 1)
							So(plans.leasesIndex, ShouldHaveLength, 0)
						})
					})

					Convey("Unleasing a plan should leave it there", func() {
						plans.Unlease(ltok)
						So(plans.queue, ShouldHaveLength, 2)
						So(plans.commissionIndex, ShouldHaveLength, 1)
						So(plans.leasesIndex, ShouldHaveLength, 0)

						Convey("Finishing it afterward should no-op", func() {
							plans.Finish(ltok)
							So(plans.queue, ShouldHaveLength, 2)
							So(plans.commissionIndex, ShouldHaveLength, 1)
							So(plans.leasesIndex, ShouldHaveLength, 0)
						})
					})
				})
			})
		})

		Convey("Given a knowledge base with a commission missing catalogs", func() {
			kb := cassandra_mem.New()
			kb.PublishCommission(cmsh_xilence)
			kb.PublishCatalog(cat_apollo1) // one of the two deps

			Convey("Foreman plans no formulas because it can't satisfy deps", func() {
				// these are basically "it shouldn't blow up tests".
				mgr := &Foreman{cassy: kb}
				mgr.register()
				pumpn(mgr, 1)

				So(mgr.currentPlans.queue, ShouldHaveLength, 0)

				Convey("Adding the missing catalog, but empty, still triggers nothing", func() {
					kb.PublishCatalog(cat_falsetto0)
					pumpn(mgr, 1)

					So(mgr.currentPlans.queue, ShouldHaveLength, 0)
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
