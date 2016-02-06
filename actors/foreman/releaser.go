package foreman

import (
	"polydawn.net/repeatr/model/cassandra"
	"polydawn.net/repeatr/model/catalog"
	"polydawn.net/repeatr/model/formula"
)

func makeReleases(kb cassandra.Cassandra, p *plan, results *formula.Stage3) []*catalog.Book {
	newEditions := make([]*catalog.Book, 0)

	// Check each output: is it proposed for releasing?
	for outName, out := range results.Outputs {
		// placeholder behavior for PoC: evvverything is release-destined!
		// invent catalog names based on the commissionID and output name.
		catID := catalog.ID(string(p.commissionedBy) + "::" + string(outName))

		// get existing catalog state so we can append
		cat := kb.Catalog(catID)
		if cat == nil {
			cat = catalog.New(catID)
		}

		// append our new products to each appropriate track
		// for PoC, always just the default track.
		trackNames := []string{""}
		for _, trackName := range trackNames {
			cat = cat.Release(
				trackName, catalog.SKU{Type: out.Type, Hash: out.Hash},
			)
		}

		// append to the list of exciting new results
		newEditions = append(newEditions, cat)
	}

	// return list of all new catalogs.  caller can publish these to the kb.
	return newEditions
}
