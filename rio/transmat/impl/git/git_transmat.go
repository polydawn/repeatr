package git

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/rio"
)

const Kind = rio.TransmatKind("git")

var _ rio.Transmat = &GitTransmat{}

type GitTransmat struct {
	workArea workArea
}

var _ rio.TransmatFactory = New

func New(workPath string) rio.Transmat {
	mustDir(workPath)
	workPath, err := filepath.Abs(workPath)
	if err != nil {
		panic(rio.TransmatError.New("Unable to set up workspace: %s", err))
	}
	wa := workArea{
		fullCheckouts:  filepath.Join(workPath, "full"),
		nosubCheckouts: filepath.Join(workPath, "nosub"),
		gitDirs:        filepath.Join(workPath, "gits"),
	}
	mustDir(wa.fullCheckouts)
	mustDir(wa.nosubCheckouts)
	mustDir(wa.gitDirs)
	return &GitTransmat{wa}
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
	try.Do(func() {
		// Basic validation and config
		//config := rio.EvaluateConfig(options...)
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Short circut out if we have the whole hash cached.
		finalPath := t.workArea.getFullchFinalPath(string(dataHash))
		if _, err := os.Stat(finalPath); err == nil {
			arena.workDirPath = finalPath
			arena.hash = dataHash
			return
		}

		// Emit git version.
		// Until we get a reasonably static version linked&contained, this is going to be an ongoing source of potential trouble.
		gitv := git.Bake("version").CombinedOutput()
		log.Info("using `git version`:", "v", strings.TrimSpace(gitv))

		// Ping silos
		if len(siloURIs) < 1 {
			panic(rio.ConfigError.New("Materialization requires at least one data source!"))
			// Note that it's possible a caching layer will satisfy things even without data sources...
			//  but if that was going to happen, it already would have by now.
		}
		// Our policy is to take the first path that exists.
		//  This lets you specify a series of potential locations,
		//  and if one is unavailable we'll just take the next.
		var warehouse *Warehouse
		for _, uri := range siloURIs {
			wh := NewWarehouse(uri)
			pong := wh.Ping()
			if pong == nil {
				log.Info("git: connected to remote warehouse", "remote", uri)
				warehouse = wh
				break
			} else {
				log.Info("Warehouse unavailable, skipping",
					"remote", uri,
					"reason", pong.Message(),
				)
			}
		}
		if warehouse == nil {
			panic(rio.WarehouseUnavailableError.New("No warehouses were available!"))
		}
		gitDirPath := t.workArea.gitDirPath(warehouse.url)

		// Fetch objects.
		func() {
			started := time.Now()
			yank(
				log,
				gitDirPath,
				warehouse.url,
			)
			log.Info("git: fetch complete",
				"elapsed", time.Now().Sub(started).Seconds(),
			)
		}()

		// Enumerate and fetch submodule objects.
		submodules := listSubmodules(string(dataHash), gitDirPath)
		log.Info("git: submodules found",
			"count", len(submodules),
		)
		submodules = applyGitmodulesUrls(string(dataHash), gitDirPath, submodules)
		func() {
			started := time.Now()
			for _, subm := range submodules {
				// Skip yank if we have the full checkout cached already.
				if _, err := os.Stat(t.workArea.getNosubchFinalPath(subm.hash)); err == nil {
					continue
				}
				// Okay, we need more stuff.  Fetch away.
				yank(
					log.New("submhash", subm.hash),
					t.workArea.gitDirPath(subm.url),
					subm.url,
				)
			}
			log.Info("git: fetch submodules complete",
				"elapsed", time.Now().Sub(started).Seconds(),
			)
		}()

		// Checkout.
		// Pick tempdir under full checkouts area.
		// We'll move from this tmpdir to the final one after both of:
		//  - this checkout
		//  - AND getting all submodules in place
		arena.workDirPath = t.workArea.makeFullchTempPath(string(dataHash))
		defer os.RemoveAll(arena.workDirPath)
		func() {
			started := time.Now()
			checkout(
				log,
				arena.workDirPath,
				string(dataHash),
				gitDirPath,
			)
			log.Info("git: checkout main repo complete",
				"elapsed", time.Now().Sub(started).Seconds(),
			)
		}()

		// Checkout submodules.
		// Pick tempdirs under the no-sub checkouts area (because we won't be recursing on these!)
		func() {
			started := time.Now()
			for _, subm := range submodules {
				pth := t.workArea.makeNosubchTempPath(subm.hash)
				defer os.RemoveAll(pth)
				checkout(
					log.New("submhash", subm.hash),
					pth,
					subm.hash,
					t.workArea.gitDirPath(subm.url),
				)
				moveOrShrug(pth, t.workArea.getNosubchFinalPath(subm.hash))
			}
			log.Info("git: checkout submodules complete",
				"elapsed", time.Now().Sub(started).Seconds(),
			)
		}()

		// Copy in submodules.
		func() {
			started := time.Now()
			for _, subm := range submodules {
				if err := fs.CopyR(
					t.workArea.getNosubchFinalPath(subm.hash),
					filepath.Join(arena.workDirPath, subm.path),
				); err != nil {
					panic(Error.New("Unexpected issues copying between local cache layers: %s", err))
				}
			}
			log.Info("git: full work tree assembled",
				"elapsed", time.Now().Sub(started).Seconds(),
			)
		}()

		// Since git doesn't convey permission bits, the default value
		// should be 1000 (consistent with being accessible under the "routine" policy).
		// Chown/chmod everything as such.
		if err := fs.Chownr(arena.workDirPath, git_uid, git_gid); err != nil {
			panic(rio.TransmatError.New("Unable to coerce perms: %s", err))
		}

		// verify total integrity
		// actually this is a nil step; there's no such thing as "acceptHashMismatch", checkout would have simply failed
		arena.hash = dataHash

		// Move the thing into final place!
		pth := t.workArea.getFullchFinalPath(string(dataHash))
		moveOrShrug(arena.workDirPath, pth)
		arena.workDirPath = pth
		log.Info("git: repo materialize complete")
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return arena
}

func (t GitTransmat) Scan(
	kind rio.TransmatKind,
	subjectPath string,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.CommitID {
	var commitID rio.CommitID
	try.Do(func() {
		// Basic validation and config
		//config := rio.EvaluateConfig(options...)
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Get off my lawn.
		panic(errors.NotImplementedError.New("The git transmat does not support scan."))
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return commitID
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
