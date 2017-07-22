package api

import (
	"github.com/polydawn/refmt/obj/atlas"

	"go.polydawn.net/repeatr/api/rdef"
)

var Atlas = atlas.MustBuild(
	ReleaseItemID_AtlasEntry,
	Catalog_AtlasEntry,
	ReleaseEntry_AtlasEntry,
	rdef.WareID_AtlasEntry,
	Replay_AtlasEntry,
	Step_AtlasEntry,
	rdef.Formula_AtlasEntry,
	rdef.FormulaAction_AtlasEntry,
	rdef.RunRecord_AtlasEntry,
)
