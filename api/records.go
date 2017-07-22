package api

import (
	"fmt"
	"strings"

	"github.com/polydawn/refmt/obj/atlas"

	"go.polydawn.net/repeatr/api/rdef"
)

/*
	Release names come in three parts:
	the Catalog Name, the Release Name, and the Item Label.

	The catalog is a whole group of releases for a single project.
	For example, there's a "repeatr" releases catalog.

	The release name is a string attached to the release;
	it's specified by the releaser when creating a new release.

	The item name is a selector used to select a specific ware when there is
	more than one ware published in a single atomic release.
	Usually these are used to specify things like a host architecture for binaries,
	though keywords like "src" and "docs" are also common.
	The set of items in a release is considered immutable once the release is published.
	Generally, there's an expectation in the ecosystem that the set of item labels available
	from each release will be the same: e.g., when upgrading from an older version
	of repeatr, one might expect to jump from "repeatr.io/repeatr:1.0:linux-amd64"
	to "repeatr.io/repeatr:1.1:linux-amd64".
*/
type (
	CatalogName string // oft like "project.org/thing".  The first part of an identifying triple.
	ReleaseName string // oft like "1.8".  The second part of an identifying triple.
	ItemName    string // oft like "linux-amd64" or "docs".  The third part of an identifying triple.
)

type ReleaseItemID struct {
	CatalogName
	ReleaseName
	ItemName
}

func ParseReleaseItemID(x string) (v ReleaseItemID, err error) {
	ss := strings.Split(x, ":")
	switch len(ss) {
	case 3:
		v.ItemName = ItemName(ss[2])
		fallthrough
	case 2:
		v.ReleaseName = ReleaseName(ss[1])
		fallthrough
	case 1:
		v.CatalogName = CatalogName(ss[0])
		return
	default:
		return ReleaseItemID{}, fmt.Errorf("ReleaseItemIDs are a colon-separated three-tuple; no more than two colons may appear!")
	}
}

var ReleaseItemID_AtlasEntry = atlas.BuildEntry(ReleaseItemID{}).Transform().
	TransformMarshal(atlas.MakeMarshalTransformFunc(
		func(x ReleaseItemID) (string, error) {
			return string(x.CatalogName) + ":" + string(x.ReleaseName) + ":" + string(x.ItemName), nil
		})).
	TransformUnmarshal(atlas.MakeUnmarshalTransformFunc(
		func(x string) (ReleaseItemID, error) {
			ss := strings.Split(x, ":")
			return ReleaseItemID{CatalogName(ss[0]), ReleaseName(ss[1]), ItemName(ss[2])}, nil
		})).
	Complete()

/*
	A Catalog is the accumulated releases for a particular piece of software.

	A Catalog indicates a single author.  When observing new releases and/or
	metadata updates in a Catalog over time, you should expect to see it signed
	by the same key.  (Signing is not currently a built-in operation of `hitch`,
	but may be added in future releases.)
*/
type Catalog struct {
	// Name of self.
	Name CatalogName

	// Ordered list of release entries.
	// Order not particularly important, though UIs generally display in this order.
	// Most recent entries are typically placed at the top (e.g. index zero).
	//
	// Each entry must have a unique ReleaseName in the scope of its Catalog.
	Releases []ReleaseEntry
}

var Catalog_AtlasEntry = atlas.BuildEntry(Catalog{}).StructMap().Autogenerate().Complete()

type ReleaseEntry struct {
	Name     ReleaseName
	Items    map[ItemName]rdef.WareID
	Metadata map[string]string
	Hazards  map[string]string
	Replay   *Replay
}

var ReleaseEntry_AtlasEntry = atlas.BuildEntry(ReleaseEntry{}).StructMap().Autogenerate().Complete()

type Replay struct {
	// The set of steps recorded in this replay.
	// Each step contains a formula with precise instructions on how to run the step again,
	// and additional data on where the inputs were selected from, so that
	// recursive audits can work automatically.
	Steps map[StepName]Step

	// Map wiring the ItemNames in the release outputs to a step and output slot
	// within that step's formula.
	//
	// Implicitly, all the ReleaseItemID's tend to be of "wire" type.
	// (It's rare, but valid, for the Products map to point directly to other
	// catalogs.  This feature can be used for example to make a personal catalog
	// which releases already-published-elsewhere content, but with different metadata.)
	//
	// As with "wire" mode in a Step's Imports, if the referenced step has more than
	// one RunRecord, then the wired output slot MUST have resulted in the same WareID
	// hash all the RunRecords.
	// Furthermore, the WareID in those RunRecords must match the WareID which
	// is directly listed for the ItemName in the the release entry; otherwise,
	// the replay isn't describing the same thing released!
	Products map[ItemName]ReleaseItemID
}

var Replay_AtlasEntry = atlas.BuildEntry(Replay{}).StructMap().Autogenerate().Complete()

type StepName string
type Step struct {
	// Record upstream names for formula inputs.
	//
	// Each key must match an input key in the formula or it is invalid.
	// The formula may have inputs that are not explained here (though tools
	// should usually emit a warning about such unexplained blobs).
	//
	// Imports may either be the full `{CatalogName,ReleaseName,ItemName}` tuple
	// referring to another catalog, or, `{"wire",StepName,OutputSlot}`.
	//
	// In the "wire" mode, the reference is interpreted as another step in this replay.
	// Hashes coming from a "wire" may be purely internal to the replay
	// (meaning, practically speaking, that ware may be an intermediate which
	// is not actually be stored anywhere).
	// If step referred to by a "wire" has more than one RunRecord, the wired
	// output slot MUST have resulted in the same WareID hash
	// all the RunRecords, or the replay is invalid.
	Imports map[rdef.AbsPath]ReleaseItemID

	// The formula for this step, exactly as executed by the releaser.
	//
	// This includes inputs (with full hashes), the script run,
	// and the output slots saved.
	// Names of inputs are separately stored; they're in the `Import` field.
	// Results are separately stored: they're in the `RunRecords` field.
	//
	// Note: it's entirely possible for two steps with different names in a Replay
	// to have identical formulas (and thus identical setupHashes).
	// In this case, both steps may also share identical RunRecords(!), if the
	// original releaser used a formula runner smart enough to notice this
	// and dedup the computation; the steps are still stored separately, because
	// it is correct to render them separately in order to represent the
	// releaser's original intentions clearly.
	Formula *rdef.Formula

	// RunRecords from executions of this formula.
	// May be one or multiple.
	//
	// These are only the records included by the releaser at the time of release.
	// Other rebuilders may have more RunRecords to share, but these are stored
	// elsewhere (you may look them up by using the Formula.SetupHash).
	//
	// It is forbidden to have two RunRecords in the same step to declare
	// different resulting WareIDs for an output slot
	// if that output slot is referenced by either the final Products
	// or any intermediate "wire"-mode Import.
	// (It's perfectly fine for RunRecords to have differing results for outputs
	// that *aren't* so referenced; an output slot which captures logs, for example,
	// may often differ between runs, but since it's not passed forward, so be it.)
	RunRecords map[rdef.RunRecordHash]*rdef.RunRecord

	// FUTURE/REVIEW: we may want to include a concept of "checkpoints":
	// they're named in the same space as steps using formulas,
	// and can be wired roughly the same way...
	// but take a single input wire,
	// and some other step names from which it only collects exit codes,
	// and the checkpoint itself does not need a formula or runrecords.
	// The checkpoint is only considered done when all the other named steps have
	// exited success, and thus can be used to gate flow based on tests in other formulas.
	// You could emulate this with dummy formulas, but
	// Checkpoints just happen to not need execution of a formula themself,
	// and only have a single output ware (which is coincidentally also the input ware),
	// and so benefit from some directness and simplification.
	// They would also be appropriate to highlight slightly differently in a UI.
}

var Step_AtlasEntry = atlas.BuildEntry(Step{}).StructMap().Autogenerate().Complete()

/*
	A note on storage (on filesystem):

		./{catalogName}/catalog.json
		./{catalogName}/replay/{releaseName}/replay.json
		./{catalogName}/replay/{releaseName}/{stepName-1}.formula
		./{catalogName}/replay/{releaseName}/{stepName-2}.formula
		./{catalogName}/tracks.json # optional; this is an extension

	Why like this?

		- Every catalog deserves its own dir; goes without saying.
		- 'catalog.json' is just about the facts; just names and content.
		- 'replay/*', though we certainly hope you're using repeatr, is
		   full of repeatr opinions.  ('catalog.json' isn't, aside from WareIDs.)
		- Formulas *get their own file* under the replay dirs, so you can easily
		   *directly* `repeatr run` those again, even without the rest of hitch.
		- The path to formulas is entirely human names, so you can tabcomplete to
		   something that's of interest to you.
		- Extensions like "tracks" (an alternative to semver) store their data
		   off to the side; they can reference <Catalog,Release,Item> tuples clearly.

	Todo: runrecord storage not yet shown here.  Pick that (see also review
	comments in the example, below).
*/
