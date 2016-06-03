package cachedir

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/inconshreveable/log15"

	"polydawn.net/repeatr/rio"
	"polydawn.net/repeatr/rio/transmat/mux"
)

var _ rio.Transmat = &CachingTransmat{}

/*
	Proxies a Transmat (or set of dispatchable Transmats), keeping a cache of
	filesystems that are requested.

	Caching is based on CommitID.  Thus, any repeated requests for the same CommitID
	can be satisfied instantly, and this system does not have any knowledge of
	the innards of other Transmat, so it can be used with any valid Transmat.
	(Obviously, this also means this will *not* help any two Transmats magically
	do dedup on data *within* themselves at a higher resolution than full dataset
	commits, by virtue of not having that much understanding of proxied Transmats.)
	If two different Transmats happen to share the same CommitID "space" ("dir" and "tar"
	systems do, for example), then they may share a CachingTransmat; constructing
	a CachingTransmat that proxies more than one Transmat that *doesn't* share the same "space"
	is undefined and unwise.

	This caching implementation assumes that everyone's working with plain
	directories, that we can move them, and that posix semantics fly.  In return,
	it's stateless and survives daemon reboots by pure coincidence with no
	additional persistence than the normal filesystem provides.

	Filesystems returned are presumed *not* to be modified, or behavior is undefined and the
	cache becomes unsafe to use.  Use should be combined with some kind of `Placer`
	that preserves integrity of the cached filesystem.
*/
type CachingTransmat struct {
	dispatch.Transmat
	workPath string
}

func New(workPath string, transmats map[rio.TransmatKind]rio.TransmatFactory) *CachingTransmat {
	// Note that this *could* be massaged to fit the TransmatFactory signiture, but there doesn't
	//  seem to be a compelling reason to do so; there's not really any circumstance where
	//  you'd want to put a caching factory into a TransmatFactory registry as if it was a plugin.
	err := os.MkdirAll(filepath.Join(workPath, "committed"), 0755)
	if err != nil {
		panic(rio.TransmatError.New("Unable to create cacher work dirs: %s", err))
	}
	dispatchMap := make(map[rio.TransmatKind]rio.Transmat, len(transmats))
	for kind, factoryFn := range transmats {
		dispatchMap[kind] = factoryFn(filepath.Join(workPath, "stg", string(kind)))
	}
	ct := &CachingTransmat{
		*dispatch.New(dispatchMap),
		workPath,
	}
	return ct
}

func (ct *CachingTransmat) Materialize(
	kind rio.TransmatKind,
	dataHash rio.CommitID,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.Arena {
	if dataHash == "" {
		// if you can't give us a hash, we can't cache.
		// also this is almost certainly doomed unless one of your options is `AcceptHashMismatch`, but that's not ours to check.
		return ct.Transmat.Materialize(kind, dataHash, siloURIs, log, options...)
	}
	permPath := filepath.Join(ct.workPath, "committed", string(dataHash))
	_, statErr := os.Stat(permPath)
	if os.IsNotExist(statErr) {
		// TODO implement some terribly clever stateful parking mechanism, and do the real fetch in another routine.
		arena := ct.Transmat.Materialize(kind, dataHash, siloURIs, log, options...)
		// keep it around.
		// build more realistic syncs around this later, but posix mv atomicity might actually do enough.
		err := os.Rename(arena.Path(), permPath)
		if err != nil {
			if err2, ok := err.(*os.LinkError); ok &&
				err2.Err == syscall.EBUSY || err2.Err == syscall.ENOTEMPTY {
				// oh, fine.  somebody raced us to it.
				if err := os.RemoveAll(arena.Path()); err != nil {
					panic(rio.TransmatError.New("Error cleaning up cancelled cache: %s", err)) // not systemically fatal, but like, wtf mate.
				}
				return catchingTransmatArena{permPath}
			}
			panic(rio.TransmatError.New("Error commiting %q into cache: %s", err))
		}
	}
	return catchingTransmatArena{permPath}
}

type catchingTransmatArena struct {
	path string
}

func (a catchingTransmatArena) Path() string       { return a.path }
func (a catchingTransmatArena) Hash() rio.CommitID { return a.Hash() }
func (a catchingTransmatArena) Teardown()          { /* none */ }
