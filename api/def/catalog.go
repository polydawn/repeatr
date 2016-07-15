package def

/*
	Catalogs contain a bunch of names for Wares.
	Each entry in a catalog is one "release" in a "track";
	when Commissions refer to a catalog, or a specific track inside the catalog,
	more recent releases in a track will automatically replace older
	release entries when during Commission resolve / Formula update processes.

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
		      {ware:"t5kbjq", retraction: "secvuln"},
		    ]
		  }
		  "5.x": Track{
		    Releases[
		      {ware:"a3vcj3"},
		    ]
		  }
		  "4.x": Track{
		    Releases[
		      {ware:"a3vcj3", retraction: "compatbreak"},
		      {ware:"b56jql"},
		      {ware:"t5kbjq", retraction: "secvuln"},
		    ]
		  }
		}

	A couple of interesting events can be seen chronicled here:
	  - t5kbjq: The oldest stable release of the product has been marked as retracted,
	    and described as a security vuln.  It's still in the release record,
	    but the retraction means it will never be used in commission resolves again.
	  - b56jql: There was another release made to the "4.x" and "stable" track.
	    It's still in good standing in both (but at this point, no longer the
	    latest in "stable").
	  - a3vcj3: This was released in the "4.x" track... then retracted again,
	    described as a compatability break.  It's still valid on the "stable"
	    track, and was also released (presumably later ;)) on the "5.x" track.

	While the structure of a catalog records all the previously valid states,
	it doesn't really retain a chronology of the changes that got us here.
	So, looking at our example catalog above, we can infer that "a3vcj3"
	was probably the subject of two changes (once in a one fat-fingering,
	and a second time by a release maintainer ) --
	but that's not really a statement we can back up from looking at the
	catalog alone.
	There's enough info here to audit a claim that a Commission=>Formula
	resolution was in accordance with some signed Catalog state,
	and no more -- in particular, there's not enough info to see precisely
	when such a resolution should have begun to run aground on retractions.
*/
type Catalog struct {
	Tracks map[CatalogTrackName]CatalogTrack `json:"tracks"`
}

type CatalogTrackName string

type CatalogTrack struct {
	Releases []CatalogEntry `json:"releases"`
}

type CatalogEntry struct {
	Ware       Ware   `json:"ware"`
	Retraction string `json:"retraction,omitempty"`
}
