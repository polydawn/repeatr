package def

type Library struct {
	Catalogs       map[CatalogCID]LibraryEntry
	WhitelistWares map[Ware]struct{}
}

type LibraryEntry struct {
	Catalog         *Catalog
	WhitelistTracks map[CatalogTrackName]struct{}
	WhitelistWares  map[Ware]struct{}
}

func (lb *Library) InterestSet() map[Ware]struct{} {
	interestSet := make(map[Ware]struct{})
	// merge in all unattached ware whitelists.
	for ware, _ := range lb.WhitelistWares {
		interestSet[ware] = struct{}{}
	}
	// range over catalogs.
	for _, entry := range lb.Catalogs {
		// merge in all wares explicitly whitelisted/
		for ware, _ := range lb.WhitelistWares {
			interestSet[ware] = struct{}{}
		}
		// merge in tracks...
		mergeTrack := func(track CatalogTrack) {
			for _, catalogEntry := range track.Releases {
				if catalogEntry.Retraction != "" {
					continue
				}
				interestSet[catalogEntry.Ware] = struct{}{}
			}
		}
		// the whitelisted ones, if such a list; if no whitelist, all of them.
		if len(entry.WhitelistTracks) == 0 {
			for _, track := range entry.Catalog.Tracks {
				mergeTrack(track)
			}
		} else {
			for trackName, _ := range entry.WhitelistTracks {
				mergeTrack(entry.Catalog.Tracks[trackName])
			}
		}
	}
	return interestSet
}
