package api

import (
	"github.com/polydawn/refmt/obj/atlas"
)

var rioAtlasEntries = []*atlas.AtlasEntry{
	WareID_AtlasEntry,
}

var repeatrAtlasEntries = []*atlas.AtlasEntry{
	Formula_AtlasEntry,
	FormulaAction_AtlasEntry,
	RunRecord_AtlasEntry,
}

var hitchAtlasEntries = []*atlas.AtlasEntry{
	Catalog_AtlasEntry,
	ReleaseEntry_AtlasEntry,
	Replay_AtlasEntry,
	Step_AtlasEntry,
}

var RepeatrAtlas = atlas.MustBuild(
	aecat(
		rioAtlasEntries,
		repeatrAtlasEntries,
	)...,
)

var HitchAtlas = atlas.MustBuild(
	aecat(
		rioAtlasEntries,
		repeatrAtlasEntries,
		hitchAtlasEntries,
	)...,
)

func aecat(aess ...[]*atlas.AtlasEntry) (r []*atlas.AtlasEntry) {
	for _, aes := range aess {
		r = append(r, aes...)
	}
	return
}
