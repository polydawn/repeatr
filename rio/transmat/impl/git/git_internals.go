package git

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/polydawn/gosh"
	"github.com/vaughan0/go-ini"

	"go.polydawn.net/repeatr/rio"
)

/*
	Does a fetch of all objects covered by the usual refs (aka branches and tags)
	into the specified git dir.

	Maps the remotes as their escaped url, so you can pile multiple remotes
	into one repo and never fuss with collision issues.
*/
func yank(log log15.Logger, gitDir string, remoteURL string) {
	// Mkdir.  (Fine if exists.)
	if err := os.Mkdir(gitDir, 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}
	// Template out command with the right paths set.
	git := bakeGitDir(git, gitDir)
	// Init (is idempotent).
	git.Bake("init", "--bare").RunAndReport()
	// Fetch.
	started := time.Now()
	log.Info("git: object fetch starting",
		"remote", remoteURL,
	)
	git.Bake(
		"fetch", "--",
		remoteURL,
		"+refs/heads/*:refs/remotes/"+slugifyRemote(remoteURL)+"/*",
	).RunAndReport()
	log.Info("git: object fetch complete",
		"remote", remoteURL,
		"elapsed", time.Now().Sub(started).Seconds(),
	)
}

func hasCommit(commitHash string, gitDir string) bool {
	git := bakeGitDir(git, gitDir)
	buf := &bytes.Buffer{}
	p := git.Bake("rev-list", "--no-walk", commitHash,
		gosh.Opts{
			OkExit: gosh.AnyExit,
			Out:    buf,
		},
	).Run()
	// if nonzero, no such commit.
	if p.GetExitCode() != 0 {
		return false
	}
	// if zero, but not echoed the same string,
	//  - was either a ref that was resolved (which shouldn't be used here)
	//  - or was a tree or some other object hash (not a commit object)
	if buf.String() != commitHash+"\n" {
		return false
	}
	return true
}

func checkout(log log15.Logger, destPath string, commitHash string, gitDir string) {
	// Template out command with the right paths set.
	git := bakeGitDir(git, gitDir)
	git = bakeCheckoutDir(git, destPath)
	// Checkout.
	started := time.Now()
	log.Info("git: tree checkout starting")
	buf := &bytes.Buffer{}
	p := git.Bake(
		"checkout", "-f", commitHash,
		gosh.Opts{
			OkExit: gosh.AnyExit,
			Err:    buf,
			Out:    buf,
		},
	).Run()
	if bytes.HasPrefix(buf.Bytes(), []byte("fatal: reference is not a tree: ")) {
		panic(rio.DataDNE.New("hash %q not found in this repo", commitHash))
	}
	if p.GetExitCode() != 0 {
		// catchall.
		// this formatting is *terrible*, but we don't have a good formatter for using datakeys, either, so.
		// (blowing past this without too much fuss because we're going to switch error libraries later and it's going to fix this better.)
		panic(Error.New("git checkout failed.  git output:\n%s", buf.String()))
	}
	log.Info("git: tree checkout complete",
		"elapsed", time.Now().Sub(started).Seconds(),
	)
}

type submodule struct {
	url  string
	path string
	hash string
	// the config file includes a name, too, but we really have no use for it.
}

/*
	Greatly comparable with the `module_list` function you might find
	in "git//git-submodule.sh"... with less perl.

	The returned structures only have `path` and `hash` set.
	Use "applyGitmodulesUrls" to get the rest.
*/
func listSubmodules(commitHash string, gitDir string) []submodule {
	// Template out command with the right paths set.
	git := bakeGitDir(git, gitDir)
	// Get stream of file info.
	// We only want the gitlink ones, but there's no way to ask that ('-d' helps narrow it down to exclude files at least though).
	trees := bufio.NewScanner(bytes.NewBufferString(git.Bake(
		"ls-tree",
		"-rdz",
		"--", commitHash,
	).Output()))
	trees.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, 0); i >= 0 {
			// We have a full null-terminated line.
			return i + 1, data[0:i], nil
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	})
	// Parse!
	// Output format is "<mode> SP <type> SP <object> TAB <file>"
	submods := make([]submodule, 0)
	for trees.Scan() {
		row := trees.Text()
		if row[0:6] != "160000" {
			continue
		}
		itab := strings.Index(row, "\t")
		if itab < 0 {
			panic(Error.New("git ls-tree IO failed: row must have tab"))
		}
		hunks := strings.Split(row[0:itab], " ")
		if len(hunks) != 3 {
			panic(Error.New("git ls-tree IO failed: row must have four hunks"))
		}
		submods = append(submods, submodule{
			hash: hunks[2],
			path: row[itab+1:],
		})
	}
	if err := trees.Err(); err != nil {
		panic(Error.New("git ls-tree IO failed: %s", err))
	}
	return submods
}

func applyGitmodulesUrls(commitHash string, gitDir string, submodules []submodule) []submodule {
	// git config -f .gitmodules --get-regexp '^submodule\..*\.path$'
	submconf, err := ini.Load(grabFile(commitHash, ".gitmodules", gitDir))
	if err != nil {
		panic(Error.New(".gitmodules file does not parse: %s", err))
	}
	for secName, section := range submconf {
		// So this is fun to parse.
		// As long as it's under the "submodule" group, we'll listen.
		// Git's concept of the name for these sections isn't standard INI;
		// on the other hand, we fortunately *don't care*:
		// the names of these sections are potentially user specified and generally not our concern.
		// We do a join of the path sections to where we found gitlinks;
		// and that correctly answers all the questions we're concerned with.
		if !strings.HasPrefix(secName, "submodule \"") {
			continue
		}
		pth, ok := section["path"]
		if !ok {
			continue
		}
		url, ok := section["url"]
		if !ok {
			continue
		}
		for i, v := range submodules {
			if v.path == pth {
				submodules[i].url = url
				break
			}
		}
	}
	return submodules
}

func grabFile(commitHash string, filePath string, gitDir string) (buf io.Reader) {
	buf = &bytes.Buffer{}
	// we punt pretty hard on errors:
	//  if something goes wrong, we'll just presume there's no such file,
	//  and return a empty stream.
	defer func() {
		recover()
	}()
	bakeGitDir(git, gitDir).Bake("cat-file", "blob",
		commitHash+":"+filePath,
		gosh.Opts{
			In:  nil,
			Out: buf,
			Err: nil,
		},
	).Run()
	return
}
