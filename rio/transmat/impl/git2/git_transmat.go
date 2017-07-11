package git2

import (
	"os"
	"path/filepath"
	"time"

	"github.com/inconshreveable/log15"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/rio"
	"go.polydawn.net/repeatr/rio/transmat/mixins"
)

const (
	git_uid = 1000
	git_gid = 1000
)
const Kind = rio.TransmatKind("git2")
const SubmoduleRecursionDepth = 1

var _ rio.Transmat = &GitTransmat{}

type GitTransmat struct {
	workArea workArea
}

var _ rio.TransmatFactory = New

func New(workPath string) rio.Transmat {
	mustDir(workPath)
	workPath, err := filepath.Abs(workPath)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to set up workspace"},
			meep.Cause(err),
		))
	}
	wa := workArea{
		fullCheckouts:  filepath.Join(workPath, "full"),
		gitStorageDirs: filepath.Join(workPath, "gits"),
	}
	mustDir(wa.fullCheckouts)
	mustDir(wa.gitStorageDirs)
	return &GitTransmat{wa}
}

func selectWarehouse(log log15.Logger, siloURIs []rio.SiloURI) *Warehouse {
	// Our policy is to take the first path that exists.
	//  This lets you specify a series of potential locations,
	//  and if one is unavailable we'll just take the next.
	// Future work: cycle through later potential locations if one returns DNE!
	//  (Unfortunately this is tricky to implement efficiently with git commands.)
	if len(siloURIs) < 1 {
		panic(&def.ErrWarehouseUnavailable{
			Msg:    "No warehouse coords configured!",
			During: "fetch",
		})
	}
	var warehouse *Warehouse
	for _, uri := range siloURIs {
		wh := NewWarehouse(uri)
		pong := wh.Ping()
		if pong == nil {
			log.Info("git: connected to remote warehouse", "remote", wh.url)
			warehouse = wh
			break
		} else {
			log.Info("Warehouse unavailable, skipping",
				"remote", uri,
				"reason", pong,
			)
		}
	}
	if warehouse == nil {
		panic(&def.ErrWarehouseUnavailable{
			Msg:    "No warehouses responded!",
			During: "fetch",
		})
	}
	return warehouse
}

/*
	Git transmats plonk down the contents of one commit (or tree) as a filesystem.

	A fileset materialized by git does *not* include the `.git` dir by default,
	since those files are not themselves part of what's described by the hash.

	Git effectively "filters" out several attributes -- permissions are only loosely
	respected (execution only), file timestamps are undefined, uid/gid bits
	are not tracked, xattrs are not tracked, etc.  If you desired defined values,
	*you must still configure materialization to use a filter* (particularly for
	file timestamps, since they will otherwise be allowed to vary from one
	materialization to the next(!)).

	Git also allows for several other potential pitfalls with lossless data
	transmission: git cannot transmit empty directories.  This can be a major pain.
	Typical workarounds include creating a ".gitkeep" file in the empty directory.
	Gitignore files may also inadventantly cause trouble.  Transmat.Materialize
	will act *consistently*, but it does not overcome these issues in git
	(doing so would require additional metadata or protocol extensions).

	This transmat is *not* currently well optimized, and should generally be assumed
	to be re-cloning on all materializations -- specifically, it is not smart
	enough to recognize requests for different commits and trees from the
	same repos in order to save reclones.
*/
func (t *GitTransmat) Materialize(
	kind rio.TransmatKind,
	dataHash rio.CommitID,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.Arena {
	var arena gitArena
	meep.Try(func() {
		// Basic validation and config
		mixins.MustBeType(Kind, kind)

		// Short circut out if we have the whole hash cached.
		finalPath := t.workArea.getFullCheckoutFinalPath(string(dataHash))
		if _, err := os.Stat(finalPath); err == nil {
			arena.workDirPath = finalPath
			arena.hash = dataHash
			return
		}

		warehouse := selectWarehouse(log, siloURIs)

		arena.workDirPath = t.workArea.makeFullCheckoutTempPath(string(dataHash))
		defer os.RemoveAll(arena.workDirPath)

		log.Info("git: clone starting",
			"remote", warehouse.url,
		)

		{
			started := time.Now()
			gitClone(log, warehouse.url, CommitId2Hash(dataHash), t.workArea.gitStorageDirs, arena.workDirPath, SubmoduleRecursionDepth)
			log.Info("git: clone complete",
				"remote", warehouse.url,
				"elapsed", time.Since(started).Seconds(),
			)
		}

		// Since git doesn't convey permission bits, the default value
		// should be 1000 (consistent with being accessible under the "routine" policy).
		// Chown/chmod everything as such.
		if err := fs.Chownr(arena.workDirPath, git_uid, git_gid); err != nil {
			panic(meep.Meep(
				&rio.ErrInternal{Msg: "Unable to coerce perms"},
				meep.Cause(err),
			))
		}

		// verify total integrity
		// actually this is a nil step; there's no such thing as "acceptHashMismatch", checkout would have simply failed
		arena.hash = dataHash

		// Move the thing into final place!
		pth := t.workArea.getFullCheckoutFinalPath(string(dataHash))
		moveOrShrug(arena.workDirPath, pth)
		arena.workDirPath = pth
		log.Info("git: repo materialize complete")
	}, rio.TryPlanWhitelist)
	return arena
}

func (t GitTransmat) Scan(
	kind rio.TransmatKind,
	subjectPath string,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.CommitID {
	// Git commits would be an oddity to generate.
	//  Git trees?  Sure: a consistent result can be generated given a file tree.
	//  Git *commits*?  Not so: the "parents" info is required, and that doesn't
	//  match how we think of the world very much at all.
	panic(&def.ErrConfigValidation{
		Msg: "saving with the git transmat is not supported",
	})
}

type gitArena struct {
	workDirPath string
	hash        rio.CommitID
}

func (a gitArena) Path() string {
	return a.workDirPath
}

func (a gitArena) Hash() rio.CommitID {
	return a.hash
}

// The git transmat teardown method is a stub.
// Unlike most other transmats, this one does its own caching and does not expect
// to have another dircacher layer wrapped around it.
func (a gitArena) Teardown() {
}
