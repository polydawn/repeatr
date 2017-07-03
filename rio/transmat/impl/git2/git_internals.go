package git2

import (
	"io"
	stdioutil "io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"go.polydawn.net/meep"
	"go.polydawn.net/repeatr/rio"

	"github.com/inconshreveable/log15"
	"gopkg.in/src-d/go-billy.v3"
	"gopkg.in/src-d/go-billy.v3/osfs"
	// sdgit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/filemode"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

type filePlacer interface {
	Place(*object.File) error
}

type gitFilePlacer struct {
	billy.Filesystem
}

func (g *gitFilePlacer) Place(f *object.File) error {
	return checkoutFile(f, g)
}

func newFilePlacer(fs billy.Filesystem) filePlacer {
	return &gitFilePlacer{fs}
}

func placeTree(tree *object.Tree, fp filePlacer) {
	fileIterator := tree.Files()
	err := fileIterator.ForEach(fp.Place)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to place git tree onto the file system"},
			meep.Cause(err),
		))
	}
}

func gitLsRemote(url string) (memory.ReferenceStorage, error) {
	endpoint, err := transport.NewEndpoint(url)
	if err != nil {
		return nil, err
	}
	gitClient, err := client.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	gitSession, err := gitClient.NewUploadPackSession(endpoint, nil)
	if err != nil {
		return nil, err
	}
	advertisedRefs, err := gitSession.AdvertisedReferences()
	if err != nil {
		return nil, err
	}
	refs, err := advertisedRefs.AllReferences()
	if err != nil {
		return nil, err
	}
	err = gitSession.Close()
	if err != nil {
		return nil, err
	}
	return refs, nil
}

func gitCheckout(commit *object.Commit, fs billy.Filesystem) {
	tree, err := commit.Tree()
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Commit missing tree object"},
			meep.Cause(err),
		))
	}
	placeTree(tree, newFilePlacer(fs))
}

/*
	Transform the rio commit ID to a git hash.
	Performs some basic checks on inputs.
*/
func CommitId2Hash(hash rio.CommitID) plumbing.Hash {
	mustBeFullHash(hash)
	ref := plumbing.NewReferenceFromStrings("", string(hash))
	return ref.Hash()
}

/*
	Reads the gitmodules file and checks each entry against the commit tree entries.
	Returns a map of submodule config objects to matching tree entries
*/
func listSubmodules(commit *object.Commit, fs billy.Filesystem, commitHash plumbing.Hash) map[*config.Submodule]*object.TreeEntry {
	tree, err := commit.Tree()
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Commit missing tree object"},
			meep.Cause(err),
		))
	}

	cfgModules, err := readGitmodulesFile(fs)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to read gitmodules file"},
			meep.Cause(err),
		))
	}

	result := map[*config.Submodule]*object.TreeEntry{}
	if cfgModules != nil {
		for _, submodule := range cfgModules.Submodules {
			if submodule == nil {
				panic(meep.Meep(
					&rio.ErrInternal{Msg: "nil submodule in list"},
				))
			}
			entry, err := tree.FindEntry(submodule.Path)
			if err != nil {
				panic(meep.Meep(
					&rio.ErrInternal{Msg: "Failed to match submodule entry to tree entry"},
					meep.Cause(err),
				))
			}
			isSubmodule := entry.Mode == filemode.Submodule
			if !isSubmodule {
				panic(meep.Meep(
					&rio.ErrInternal{Msg: "gitmodule entry is not a submodule"},
				))
			}
			result[submodule] = entry
		}
	}
	return result
}

/*
	Clones the remote repository to workingDirectory.
	All .git directories will be cached per repository URL in cacheDir
	submoduleRecursionDepth will checkout submodule repositories up to the given depth.
		A value less than 0 implies infinite depth.
*/
func gitClone(log log15.Logger, remote string, commitHash plumbing.Hash, cacheDir string, workingDirectory string, submoduleRecursionDepth int) {
	if commitHash.IsZero() {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Commit hash may not be a zero-value"},
		))
	}

	// cache of the .git files
	cacheFS := osfs.New(filepath.Join(cacheDir, slugifyRemote(remote), commitHash.String()))
	gitStore, err := filesystem.NewStorage(cacheFS) // store git objects
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Failed to initiate git cache storage"},
			meep.Cause(err),
		))
	}

	// where the repository files will go
	fs := osfs.New(workingDirectory)

	// Check to see if the commit is cached
	commit, err := object.GetCommit(gitStore, commitHash)
	if err != nil {
		log.Info("git: object fetch starting",
			"remote", remote,
		)
		fetchStarted := time.Now()
		uploadRequest := packp.NewUploadPackRequest()
		uploadRequest.Wants = []plumbing.Hash{commitHash}
		if uploadRequest.IsEmpty() {
			panic(meep.Meep(
				&rio.ErrInternal{Msg: "Empty upload-pack-request"},
			))
		}
		gitFetch(remote, gitStore, uploadRequest)
		commit, err = object.GetCommit(gitStore, commitHash)
		log.Info("git: object fetch complete",
			"remote", remote,
			"elapsed", time.Since(fetchStarted).Seconds(),
		)

	}
	checkoutStarted := time.Now()
	log.Info("git: tree checkout starting")
	gitCheckout(commit, fs)
	log.Info("git: tree checkout complete",
		"elapsed", time.Now().Sub(checkoutStarted).Seconds(),
	)
	if submoduleRecursionDepth == 0 {
		return
	}

	submoduleStarted := time.Now()
	subs := listSubmodules(commit, fs, commitHash)
	log.Info("git: submodules found",
		"count", len(subs),
	)
	for cfg, entry := range subs {
		gitClone(
			log.New("submhash", entry.Hash),
			cfg.URL,
			entry.Hash,
			cacheDir,
			filepath.Join(workingDirectory, cfg.Path),
			submoduleRecursionDepth-1,
		)
	}
	log.Info("git: fetch submodules complete",
		"elapsed", time.Now().Sub(submoduleStarted).Seconds(),
	)
}

/*
	We force https when talking to github because github may refuse to respond to http urls.
	Otherwise this will return an endpoint that will use the correct protocol based on the remote.
	We could improve behavior by overriding installing new protocols.
	Perhaps using a particular git user agent would avoid the strange github behavior.
*/
func gitCreateEndpoint(remote string) transport.Endpoint {
	endpoint, err := transport.NewEndpoint(remote)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Failed to initiate git endpoint"},
			meep.Cause(err),
		))
	}
	if endpoint.Protocol() == "http" {
		parsedUrl, err := url.Parse(remote)
		if err != nil {
			panic(meep.Meep(
				&rio.ErrInternal{Msg: "Failed to parse git remote url"},
				meep.Cause(err),
			))
		}
		// Force https urls on github.com because github is silly and will not send back a response
		if HasFoldedSuffix(parsedUrl.Hostname(), githubHostname) {
			parsedUrl.Scheme = "https"
		}
		endpoint, err = transport.NewEndpoint(parsedUrl.String())
		if err != nil {
			panic(meep.Meep(
				&rio.ErrInternal{Msg: "Failed to initiate git endpoint"},
				meep.Cause(err),
			))
		}
	}
	return endpoint
}

/*
	Executes an upload-pack-request on the remote
	Loads the response into the provided storage
*/
func gitFetch(remote string, gitStore storer.Storer, uploadRequest *packp.UploadPackRequest) {
	endpoint := gitCreateEndpoint(remote)
	gitClient, err := client.NewClient(endpoint)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Failed to create git client"},
			meep.Cause(err),
		))
	}
	session, err := gitClient.NewUploadPackSession(endpoint, nil)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Failed to create git upload-pack session"},
			meep.Cause(err),
		))
	}
	response, err := session.UploadPack(uploadRequest)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "git upload-pack-request failed"},
			meep.Cause(err),
		))
	}
	err = packfile.UpdateObjectStorage(gitStore, response)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Failed to update git object storage"},
			meep.Cause(err),
		))
	}
	// This operation may block if the response has not been processed.
	err = session.Close()
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Failed to close upload-pack-session"},
			meep.Cause(err),
		))
	}
}

// Copied from go-git/worktree with minor changes
func checkoutFile(f *object.File, fs billy.Filesystem) (err error) {
	mode, err := f.Mode.ToOSFileMode()
	if err != nil {
		return
	}

	if mode&os.ModeSymlink != 0 {
		return checkoutFileSymlink(f, fs)
	}

	from, err := f.Reader()
	if err != nil {
		return
	}

	defer ioutil.CheckClose(from, &err)

	to, err := fs.OpenFile(f.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return
	}

	defer ioutil.CheckClose(to, &err)

	_, err = io.Copy(to, from)
	return
}

// Copied from go-git/worktree with minor changes
func checkoutFileSymlink(f *object.File, fs billy.Filesystem) (err error) {
	from, err := f.Reader()
	if err != nil {
		return
	}

	defer ioutil.CheckClose(from, &err)

	bytes, err := stdioutil.ReadAll(from)
	if err != nil {
		return
	}

	err = fs.Symlink(string(bytes), f.Name)
	return
}

// Copied from go-git/worktree with minor changes
func readGitmodulesFile(fs billy.Filesystem) (*config.Modules, error) {
	f, err := fs.Open(gitmodulesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	input, err := stdioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	m := config.NewModules()
	return m, m.Unmarshal(input)
}
