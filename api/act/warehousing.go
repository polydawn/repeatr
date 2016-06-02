package act

import (
	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/rio"
)

type Mirror func(
	destTransKind rio.TransmatKind,
	destWarehouseCoords def.WarehouseCoord,
	interestSet def.Library,
)
