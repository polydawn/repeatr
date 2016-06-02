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
