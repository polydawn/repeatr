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

/*
	WarehouseAddr strings describe a protocol and dial address for talking to
	a storage warehouse.

	The serial format is an opaque string, though they typically resemble
	(and for internal use, are parsed as) URLs.
*/
type WarehouseAddr string

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
