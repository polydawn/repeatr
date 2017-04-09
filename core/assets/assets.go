/*
	`assets` is a helper package for materializing large assets
	and plugins for repeatr's internal usage.
*/
package assets

import (
	"os"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/jank"
	"go.polydawn.net/repeatr/rio"
	"go.polydawn.net/repeatr/rio/transmat/impl/cachedir"
	"go.polydawn.net/repeatr/rio/transmat/impl/tar"
)

var assets = map[string]rio.CommitID{
	"runc": rio.CommitID("xKFg25CWzvNLqmAGkAKY_TG5dYqeJGxP7XnVYBJOQl5ZmwY1JiJc2Fl9pobqS5fW"),
}

func WarehouseCoords() []rio.SiloURI {
	return append(
		PreferredWarehouseCoords(),
		rio.SiloURI("http+ca://repeatr.s3.amazonaws.com/assets/"),
	)
}

// FIXME silly API, seealso comments in `def.WarehouseCoords`; refactor of def package will obliviate this function
func WarehouseCoords2() def.WarehouseCoords {
	wcs := make(def.WarehouseCoords, 0, 2)
	for _, x := range WarehouseCoords() {
		wcs = append(wcs, def.WarehouseCoord(x))
	}
	return wcs
}

func PreferredWarehouseCoords() []rio.SiloURI {
	val := os.Getenv("REPEATR_ASSETS")
	if val != "" {
		return []rio.SiloURI{
			rio.SiloURI(val),
		}
	}
	return nil
}

/*
	Gets a path to the rootfs of the named asset.  The asset
	may be fetched if it's not available.

	Usage might be like

		cmd.Path = filepath.Join(assets.Get("runc"), "bin/runc")

	There is no versioning information in parameters because
	this is where the buck stops: a build of repeatr was tested against
	and shall use exactly one known version of a thing.  The assets
	dirs will themselves be treated like CAS, of course: multiple
	installs of different versions of repeatr on the system may share
	an assets cache without fuss.
*/
func Get(assetName string) string {
	var arena rio.Arena
	meep.Try(func() {
		arena = transmat().Materialize(
			rio.TransmatKind("tar"),
			assets[assetName],
			WarehouseCoords(),
			log15.New(log15.DiscardHandler), // this is foolish, but i just feel Wrong requiring a logger as an arg to `asset.Get`.
		)
	}, meep.TryPlan{
		{CatchAny: true,
			Handler: meep.TryHandlerMapto(&ErrLoadingAsset{})},
	})
	return arena.Path()
}

/*
	A separate transmat is used for the asset system.

	Assets use a separate cache.

	There's also only one kind of transmat enabled here -- it really
	only makes sense for the asset system to use the tar transmat,
	since that's so easily bundled without large extraneous components.
*/
func transmat() rio.Transmat {
	workDir := filepath.Join(jank.Base(), "assets")
	dirCacher := cachedir.New(filepath.Join(workDir, "cache"), map[rio.TransmatKind]rio.TransmatFactory{
		rio.TransmatKind("tar"): tar.New,
	})
	return dirCacher
}

type ErrLoadingAsset struct {
	meep.TraitAutodescribing
	meep.TraitCausable
}
