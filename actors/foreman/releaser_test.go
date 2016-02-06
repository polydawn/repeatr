package foreman

import (
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/model/cassandra/impl/mem"
	"polydawn.net/repeatr/model/formula"
)

func TestReleasing(t *testing.T) {
	Convey("Releasing should be awesome", t, func(c C) {
		kb := cassandra_mem.New()

		Convey("A result with no outputs proposes no releases", func() {
			newEditions := makeReleases(
				kb,
				&plan{},
				(*formula.Stage3)(&def.Formula{}),
			)

			So(newEditions, ShouldHaveLength, 0)
		})

		Convey("A result with outputs proposes new catalogs", func() {
			newEditions := makeReleases(
				kb,
				&plan{},
				(*formula.Stage3)(&def.Formula{
					Outputs: def.OutputGroup{
						"coquet": &def.Output{Hash: "c1"},
						"danish": &def.Output{Hash: "d1"},
					},
				}),
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
	})
}
