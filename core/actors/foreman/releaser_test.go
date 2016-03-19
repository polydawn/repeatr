package foreman

import (
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/core/model/cassandra/impl/mem"
	"polydawn.net/repeatr/core/model/catalog"
	"polydawn.net/repeatr/core/model/formula"
	"polydawn.net/repeatr/def"
)

func TestReleasing(t *testing.T) {
	Convey("Releasing should be awesome", t, func(c C) {
		kb := cassandra_mem.New()

		Convey("A result with no outputs proposes no releases", func() {
			newEditions := makeReleases(
				kb,
				&plan{},
				&formula.Stage3{},
			)

			So(newEditions, ShouldHaveLength, 0)
		})

		Convey("A result with outputs proposes new catalogs", func() {
			newEditions := makeReleases(
				kb,
				&plan{},
				&formula.Stage3{
					Outputs: def.OutputGroup{
						"coquet": &def.Output{Hash: "c1"},
						"danish": &def.Output{Hash: "d1"},
					},
				},
			)

			So(newEditions, ShouldHaveLength, 2)
			gatheredLatestHashes := []string{
				newEditions[0].Latest().Hash,
				newEditions[1].Latest().Hash,
			}
			sort.Strings(gatheredLatestHashes)
			So(gatheredLatestHashes, ShouldResemble, []string{
				"c1",
				"d1",
			})
		})

		Convey("Given a knowledgebase with some existing catalogs", func() {
			kb.PublishCatalog(catalog.New(catalog.ID("elate::emu")).
				Release("", catalog.SKU{Hash: "e1"}),
			)

			Convey("Proposed new catalogs include prior states", func() {
				newEditions := makeReleases(
					kb,
					&plan{commissionedBy: "elate"},
					&formula.Stage3{
						Outputs: def.OutputGroup{
							"emu": &def.Output{Hash: "e2"},
						},
					},
				)

				// First of all, new catalogs should still have the hot stuff
				So(newEditions, ShouldHaveLength, 1)
				So(newEditions[0].Latest().Hash, ShouldEqual, "e2")
				// Also, the new one should to have the existing history
				So(newEditions[0].Tracks[""], ShouldHaveLength, 2)
				So(newEditions[0].Tracks[""][0].Hash, ShouldEqual, "e1")
				So(newEditions[0].Tracks[""][1].Hash, ShouldEqual, "e2")
			})
		})

	})
}
