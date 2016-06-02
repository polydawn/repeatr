package def

/*
	Catalogs contain a bunch of names for Wares.
	Each entry in a catalog is one "release" in a "track";
	when Commissions refer to a catalog, or a specific track inside the catalog,
	more recent releases in a track will automatically replace older
	release entries when during Cmomission resolve / Formula update processes.

	Tracks can be used to name a series of releases for the purpose of smooth
	updating; e.g. the "4.x" series vs the "5.x" series would be two different
	tracks.  Tracks bundled within the same Catalog share an author (and
	signing authority).

	Here's a sketch of an example catalog structure.
	Note that it's perfectly normal for different tracks to refer to the same ware.

		Catalog{
		  "stable": Track{
		    Releases[
		      {ware:"a3vcj3"},
		      {ware:"b56jql"},
		    ]
		  }
		  "4.x": Track{
		    Releases[
		      {ware:"b56jql"},
		    ]
		  }
		}
*/
type Catalog struct {
	Tracks map[CatalogTrackName]CatalogTrack `json:"tracks"`
}

/*
	Catalog IDs are claim-type structures.  You pick a secret, mix in
	a chosen name, and derive the CID from that --
	and that's what you publish under.
*/
type CatalogCID string

type CatalogTrackName string

type CatalogTrack struct {
	Releases []CatalogEntry `json:"releases"`
}

type CatalogEntry struct {
	Ware       Ware   `json:"ware"`
	Retraction string `json:"retraction,omitempty"`
}
