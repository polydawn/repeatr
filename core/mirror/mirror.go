package mirror

import (
	"polydawn.net/repeatr/api/act"
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/rio"
)

var _ act.Mirror = (&MirrorConfig{}).Mirror

type MirrorConfig struct {
	Transmat rio.Transmat // probably a dispatch but whatever
}

func (mcfg *MirrorConfig) Mirror(
	destTransKind rio.TransmatKind,
	destWarehouseCoords def.WarehouseCoord,
	library def.Library,
	otherSrcs def.WarehouseCoords,
) {
	interestSet := library.InterestSet()
	for ware := range interestSet {
		// check for dest presence; skip out if so (optionally, perhaps validate?)
		// fetch from somewhere, pipe to dest (can we do that without a tmpfs?)
		_ = ware
	}
}
