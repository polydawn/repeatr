package localforeman

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"polydawn.net/repeatr/model/cassandra"
	"polydawn.net/repeatr/model/catalog"
)

func Test(t *testing.T) {
	Convey("Given a small knowledge base", t, func(c C) {
		kb := cassandra.New()
		// publish artifact "apollo" -- default track only, single release
		kb.PublishCatalog(&catalog.Book{
			catalog.ID("apollo"),
			map[string][]catalog.SKU{"": []catalog.SKU{
				{"tar", "a1"},
			}},
		})
		// publish artifact "balogna" -- default track only, two releases
		kb.PublishCatalog(&catalog.Book{
			catalog.ID("apollo"),
			map[string][]catalog.SKU{"": []catalog.SKU{
				{"tar", "b1"},
				{"tar", "b2"},
			}},
		})

		Convey("Formulas are emitted for all plans using latest editions of catalogs", func() {
			// TODO fragile; desire a way to check if we can stop pumping :/ don't want to block in event of bugs

			// should also be able to test an *empty* knowledge base produces no action but doesn't crash (!)
		})
	})
}
