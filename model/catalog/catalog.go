package catalog

// catalogID will probably grow to this whole fancy struct with sigs and stuff
//  and mime half of an SFS filesystem.  #notwithinfirstdraft

// placeholder...?  we'll later need structure for verifying WoT-y ID, but this ain't it; this needs to be a simple fast map key.
type ID string

// catalog should probably be flipped to a concrete type.

type Book struct {
	ID ID

	Tracks map[string][]SKU

	// Totally accounted for and awesome in this design:
	//  - Tracks (e.g. subscribing to the "4.x" series of "gcc" but not the "5.x").
	//        We did it.  Tracks are a thing.
	//        The blank string is the name of the default track.  The default
	//      track is required.  All others are optional.
	//        It would be possible to do this from one layer out, but I think
	//      the cognitive and maintence overhead would be damaging -- mostly
	//      because catalogs are shaping up to be our fundamental unit of naming,
	//      and people really do want "gcc" to be a name, and not "gcc-5".
	//      Similarly, the signing keys for those things had better dang well be
	//      the same.
	//        Building named tracks which must themselves be linear does
	//      still seem like a good idea for sidesteping version nomenclature
	//      wars entirely.  Want "stable" and "testing" tracks?  Fine.  Want
	//      a track for every "Major.Minor" of your psuedo-semver?  Fine.
	//      (Honestly, I kind of like the impact this might have on version
	//      stability and naming conventions.  "Did you promise this would
	//      have nonbreaking API changes [Y/N]" in a very literal way seems
	//      much more clear and empowering than dealing with the cultural
	//      tidal zones that result when people put "~>3.6.1" in configs.)
	//
	// Not yet accounted for in this design:
	//  - Aliasing (same data kept in different kinds of warehouse, resulting in different ids).
	//        Not clear how the caller would pick the most preferred type, since the
	//      caller is generally expected to be headless itself.  Not at all clear
	//      how to handle aliasing in general, really, since it's hard to create
	//      immutable tokens that are future-proof.
	//        Requiring preference-ranked list as a parameter could resolve
	//      ambiguity, but it's unclear how the caller would get great mileage
	//      out of that either: e.g. do you want the "dir" answer or the "git"
	//      answer, well, depends on which one I have that's closer (well, or
	//      which part of my universe I'm trying to explore, in which case it's
	//      much clearer).
	//        This is not a new problem; transmats already have (and kick the can on)
	//      exactly this problem with future-proofed-porting.  So, perhaps if
	//      that can't be resolved there, it's just as well to continue punting
	//      here, and let it be resolved yet one more layer further up: with
	//      future porting created by another layer of (as yet unnecessary) namespaces.
	//  - Flags and tags (marking something "insecure" or "unsupported" without delisting it).
	//        Not sure if this can be done without causing headaches.
	//      Namely, if there's a tag string we don't recognize, what do --
	//      refuse to build it, or ignore it and carry on?  (Failsafe, prob.)
	//        I kind of like this because delisting means a wary consumer is
	//      going to be forced to demand explanations out of band, and that
	//      means it realistically won't usually happen and/or will result
	//      in ad-hoc solutions (which will never fully penetrate, etc).  I
	//      hesitate because I'm not convinced it passes YAGNI thresholds yet.
}

func (b *Book) Latest() SKU {
	defaultTrack := b.Tracks[""]
	n := len(defaultTrack)
	if n < 1 {
		return SKU{}
	}
	return defaultTrack[n-1]
}

/*
	Ordered list of all valid versions; latest last.

	Note: this may not be exactly the same thing as "all history ever".
	When a new edition of a catalog is published, it's certainly at
	liberty to *remove* old versions: this is like saying "no, this
	one is really out of stock: there's security issues, please stop
	ever considering using it".  (It's also valid to keep track of
	old editions of catalogs so it's possible to Raise Questions
	if something changes or disappeared unexpectedly; a catalog
	edition should be an immutable thing, and changes roughly explainable
	by a reasonable person.)
*/
func (b *Book) All() []SKU {
	return b.Tracks[""]
}

// id a la https://en.wikipedia.org/wiki/Stock-keeping_unit
type SKU struct {
	Type string `json:"type"`
	Hash string `json:"hash"`
}
