package scanCmd

import (
	"github.com/inconshreveable/log15"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor/util"
	"go.polydawn.net/repeatr/rio"
)

/*
	Returns an output specification complete with hash, which can be
	flipped around and used as an `Input` specification in a `Formula`.
*/
func scan(outputSpec def.Output, log log15.Logger) def.Output {
	// TODO validate MountPath exists, give nice errors

	// todo: create validity checking api for URIs, check them all before launching anything
	warehouses := make([]rio.SiloURI, len(outputSpec.Warehouses))
	for i, wh := range outputSpec.Warehouses {
		warehouses[i] = rio.SiloURI(wh)
	}

	// So, this CLI command is *not* in its rights to change the subject area,
	//  so take that as a pretty strong hint that filters are going to have to pass down *into* transmats.
	commitID := util.DefaultTransmat().Scan(
		// All of this stuff that's type-coercing?
		//  Yeah these are hints that this stuff should be facing data validation.
		rio.TransmatKind(outputSpec.Type),
		outputSpec.MountPath,
		warehouses,
		log,
		rio.ConvertFilterConfig(*outputSpec.Filters)...,
	)

	outputSpec.Hash = string(commitID)
	return outputSpec
}
