package api

/*
	This file is all serializable types used in Rio
	to define filesets, WareIDs, packing systems, and storage locations.
*/

import (
	"fmt"
	"strings"

	"github.com/polydawn/refmt/obj/atlas"
)

/*
	Ware IDs are content-addressable, cryptographic hashes which uniquely identify
	a "ware" -- a packed filesystem snapshot.

	Ware IDs are serialized as a string in two parts, separated by a colon --
	for example like "git:f23ae1829" or "tar:WJL8or32vD".
	The first part communicates which kind of packing system computed the hash,
	and the second part is the hash itself.
*/
type WareID struct {
	Type string
	Hash string
}

func ParseWareID(x string) (WareID, error) {
	ss := strings.SplitN(x, ":", 2)
	if len(ss) < 2 {
		return WareID{}, fmt.Errorf("wareIDs must have contain a colon character (they are of form \"<type>:<hash>\")")
	}
	return WareID{ss[0], ss[1]}, nil
}

func (x WareID) String() string {
	return x.Type + ":" + x.Hash
}

var WareID_AtlasEntry = atlas.BuildEntry(WareID{}).Transform().
	TransformMarshal(atlas.MakeMarshalTransformFunc(
		func(x WareID) (string, error) {
			return x.String(), nil
		})).
	TransformUnmarshal(atlas.MakeUnmarshalTransformFunc(
		func(x string) (WareID, error) {
			return ParseWareID(x)
		})).
	Complete()

type AbsPath string // Identifier for output slots.  Coincidentally, a path.

type (
	/*
		WarehouseAddr strings describe a protocol and dial address for talking to
		a storage warehouse.

		The serial format is an opaque string, though they typically resemble
		(and for internal use, are parsed as) URLs.
	*/
	WarehouseAddr string

	/*
		Configuration details for a warehouse.

		Many warehouses don't *need* any configuration; the addr string
		can tell the whole story.  But if you need auth or other fanciness,
		here's the place to specify it.
	*/
	WarehouseCfg struct {
		Auth     string      // auth info, if needed.  usually points to another file.
		Addr     interface{} // additional addr info, for protocols that require it.
		Priority int         // higher is checked first.
	}

	/*
		A suite of warehouses.  A transmat can take the entire set,
		and will select the ones it knows how to use, sort them,
		ping each in parallel, and start fetching from the most preferred
		one (or, from several, if it's a really smart protocol like that).
	*/
	WorkspaceWarehouseCfg map[WarehouseAddr]WarehouseCfg
)

/*
	For workspace configuration: warehouses may involve auth, so
	we definitely don't want them to require redundant declaration:

		{
			"warehouseName": {
				"auth": {...},
				"addr": "...",
				"prio": "9000", // ??
			},
		}

	Should the warehouse name in that doc simply *be* the addr?  Probably.
	(Although in some cases things may be more complicated.
	For example, for IPFS, that might be bootstrap node addr *list*;
	the warehouseName would still probably want to carry a semantic though:
	it would likely bear a prefix path, or IPNS key, or something.)

	For hitch: we don't generally want to provide auth, nor would
	prioritization overrides be sane, so, there, the warehouse name and
	addr info most definitely default to being one and the same.

	The nature of things as strings alone for hitch would seem to indicate
	that that string should be the join column for everyone:
	repeatr formulas with discovery hint sections should use that string as well.

	For formulas: ahhhhhh.  Here things get truly strange.

	For formulas, you may want to give a warehouse name hint per input path.
	Or, per hash.  (That's roughly the same thing, unless the same hash shows
	up multiple times, which is a rare coincidence we can effectively ignore.)
	Or, per import name.

	Formulas also need the warehouse names *again* for *saving outputs*.
	This is often simpler, because the most sensible user story is to upload things
	to one(!) storage (usually, a local disk...!), and mirror thereafter.
	But what if you have two outputs, one more secret than the other?
	Or two outputs, one of which is simply a totally different pack type
	(perhaps you have a trans-pack-mirror-fanout step -- hah!)?
	Sensible defaults will carry us a long way for most of the time, but
	some defined way to deal with the interesting cases is also needed.

	For formula outputs, then, we have learned something: they *definitely* need
	to be able to specify a warehouseName per path.
	There simply isn't anything else to go by!  Tags don't belong in
	formulas anymore; that's a thing we definitely know now in this
	brave new world with hitch and the clear separation of planners and execution.

	What else do we know about the story for formula inputs?

	  - increasingly, formulas are being treated as halfway to assembly: they're allowed to be a tad uggo.
	  - warehouse info still *absolutely* must go in a top level "discovery" section and be joinable, *not* in inputs.
	  - i guess it coudl be in the imports but that seems wrong, and uncongruent with how outputs could never be.
	  - imports is also a bad idea simply because repeatr itself is still not supposed to *see* imports, really.  merely be tolerant of them being in the same doc as the (rest?  no, *real*) formula.
	  - paths are a red herring, though.  aren't they?  this should be more like "have hash; inquire with db/daemons where to get it"

	But that's a SIGINT leak.

	Let's focus back to SIGINT, then, as a success heuristic.

	In short: that means some sort of naming layer needs to be the root of deciding
	what lookup attempts to make.  (The fact it may give us a chance to save ourself
	from the shittiness of the git protocol's ability to advertise available objects
	is merely "coincidental" (though admittedly also critical).)

	Addntl user story: usually, things come in *pools*.
	There's the public mirror pool of popular upstream stuff.
	There's probably going to be a rix objects pool.  Just makes sense, for budgetary reasons.
	There's probably going to be a *private* pool at your office.
	Git repos each act like their own pools and they're really shits for this.
	And you might have a *private* pool at home for your side projects or consultancy gigs or whatever.

	There's an interesting dichotomy here:
	with *public* objects, you don't really mind over-asking about them;
	with *private* objects (or git), you're quite worried about over-asking.
	(N.b. Even if you mind your SIGINT so much that you don't want to ask your $dayjob
	storage pools at phpcompany about your nodejs asset needs, you solve that by just
	*not listing* that private storage pool in that workspace.  You still have workspaces.)

	(There's also the outlier of non-CAS / single-ware warehouses.  Those are odd enough
	that they can have their own entire dang schema section; we can disregard them here.)

	So maybe this gives us leverage on the safe defaults thing.
	We just ask *all* your pools.
	Unless it's got a specific one listed.

	Voilà;
	you can give explicits for git, so we can literally even;
	you can say nothing at all by default, and get sensible results sans fuss;
	and if you're a company that's paranoid af, you can either generate quite verbose
	override-ridden documents, or, come to think of it, just *don't have* public pools
	in the first place (mirroring is easy by design, after all, so why not?).

*/

/*
	Ok but so how do these all get *stacked up*?

	User story: I have a git repo.  I ran a tool some time ago to 99% pin a formula
	(except for the repo hash, ofc).  That tool did also leave import records.

	- I have my default public workspace stuff, and my default workspace-local warehouse too.
	   These probably pick up 99% of my needs.
	- given the import records and a hitch db, I can select the hitch db's recommended warehouseAddrs.
	- given the hitch db at large, I could actually brute force search for the hash and any warehouses.  (but this would be... unusual to need to do)
	- I probably *do* in fact want to do one of these two things, because they have more present-tense info than when my formula would've been pinned.
	   (remember, gitmodules files suck: checking out an old version and discovering a repo url moved is real bad experience.)
	- My tool could have pinned recommended warehouseAddrs too.  this would help people who can't be arsed to have a hitch db (which is supposed to be a supported, if somewhat odd, way to live).
	   (we should probably do this anyway.  despite the "gitmodules suck" admonishment... *we know what fallback lists* are, so we can do this without downside.)

	So, the effective config gathers all of these things:
	the workspace-wide config entries at whatever priorities you assigned them,
	the hitchdb chips in its current opinions if it and an imports list are present,
	and a warehouseaddr list per input may *also* have been generated by the pin tool.

	Oh, on top of all this we should support rewrite rules at the workspace level.

	So repeatr's CLI, interestingly, needs to pick up on *all* of these.
	Yes indeedy, that may mean having *repeatr invoke hitch*.  (Didn't see that one coming.)
	r2k8s will need it as well (those discovery-config-discover..er... funcs will be
	pretty much identical for the two of them).
*/

/*
	Hitch dealing with this:

	  { "name": "v10.0",
	    "items": {
	      "bin": "tar:woeifjwef",
	      "src": "git:aergoiaj",
	    },
	    "mirrors": {
	      "*": [
	        "https+ca://public.repeatr.io/",
	      ],
	      "src": [
	        "git+https://github.com/asdf/akaka"
	      ],
	    }
	  }

	Note the several points this has to cover:

	  - the presense of multiple items means I might need specific warehouses for
	     some of them *individually*.
	  - we are definitely going to want git things to be items... or replays will
	     often end in sadness: git *always* needs the equiv of "private" pool treatment,
	     and when hitch saves formulas in replay info, it's sans specific warehouses,
	     so you'd better dang well be able to reconstruct it from the imports.

	Also ugh did you notice git can take https transports?  That's really going to
	complicated the "A transmat can take the entire set, and will select the ones
	it knows how to use" rule.

	This mirroring list may belong per catalog rather than per release.
	I'd imagine it's almost always going to be a constant for most of the items
	(a single CA warehouse, never changing), the src git repos also tend not to change,
	and... that's about it, except for single-ware warehouses, which... i think still
	fall into our earlier dismisall of needing their own entire goddamn schema section.
*/
