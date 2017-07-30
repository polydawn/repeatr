package api

/*
	This file is all serializable types used in Rio
	to define filesets, WareIDs, packing systems, and storage locations.
*/

import (
	"fmt"
	"strings"
	"time"

	"github.com/polydawn/refmt/obj/atlas"
)

/*
	Ware IDs are content-addressable, cryptographic hashes which uniquely identify
	a "ware" -- a packed filesystem snapshot.
	A ware contains one or more files and directories, and metadata for each.

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
	FilesetFilters define how certain filesystem metadata should be treated
	when packing or unpacking files.

		For each value:
		  If set: use that number;
		    default for pack is to flatten;
		    default for unpack is to respect packed metadata.
		  To keep during pack: set the keep bool.
		If keep is true, the value must be nil or the filter is invalid.
*/
type FilesetFilters struct {
	FlattenUID struct {
		Keep  bool    `json:"keep,omitempty"`
		Value *uint32 `json:"value,omitempty"`
	} `json:"uid"`
	FlattenGID struct {
		Keep  bool    `json:"keep,omitempty"`
		Value *uint32 `json:"value,omitempty"`
	} `json:"gid"`
	FlattenMtime struct {
		Keep  bool       `json:"keep,omitempty"`
		Value *time.Time `json:"value,omitempty"`
	} `json:"mtime"`
}
