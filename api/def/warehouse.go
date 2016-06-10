package def

/*
	A list of warehouse coordinates, as simple strings (they're serialized
	as such).

	FIXME this is really ambiguous vs `rio.SiloURI`, should probably try
	to refactor to only be one.
*/
type WarehouseCoords []WarehouseCoord

type WarehouseCoord string
