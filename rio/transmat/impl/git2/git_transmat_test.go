package git2

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polydawn/gosh"
	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/repeatr/lib/testutil"
	"go.polydawn.net/repeatr/rio"
)

// Test against "real" git
var git gosh.Command = gosh.Gosh(
	"git",
	gosh.NullIO,
	gosh.Opts{
		Env: map[string]string{
			"GIT_CONFIG_NOSYSTEM": "true",
			"HOME":                "/dev/null",
			"GIT_ASKPASS":         "/bin/true",
		},
	},
)

//func TestCoreCompliance(t *testing.T) {
//	Convey("Spec Compliance: Git Transmat", t, testutil.WithTmpdir(func() {
//		// Nope.
//		// Most of the core compliance tests require round-trip;
//		// we can't satisfy any of those because we don't (yet) support scan with git.
//	}))
//}

func TestGitLocalFileInputCompat(t *testing.T) {
	// note that this test eschews use of regular file fixtures for a few reasons:
	//  - because it's capable of working without root if it doesn't try to chown
	//  - because we're doing custom content anyway so we have multiple commits
	//  both of these could be addressed with upgrades to filefixtures in the future.
	Convey("Given a local git repo", t, testutil.Requires(
		testutil.RequiresRoot,
		testutil.WithTmpdir(func(c C) {
			git := git.Bake(gosh.Opts{Env: map[string]string{
				"GIT_AUTHOR_NAME":     "repeatr",
				"GIT_AUTHOR_EMAIL":    "repeatr",
				"GIT_COMMITTER_NAME":  "repeatr",
				"GIT_COMMITTER_EMAIL": "repeatr",
			}})
			var dataHash_1 rio.CommitID
			var dataHash_2 rio.CommitID
			var dataHash_3 rio.CommitID
			var dataHash_4 rio.CommitID
			git.Bake("init", "--", "repo-a").RunAndReport()
			testutil.UsingDir("repo-a", func() {
				git.Bake("commit", "--allow-empty", "-m", "testrepo-a initial commit").RunAndReport()
				dataHash_1 = rio.CommitID(strings.Trim(git.Bake("rev-parse", "HEAD").Output(), "\n"))
				ioutil.WriteFile("file-a", []byte("abcd"), 0644)
				git.Bake("add", ".").RunAndReport()
				git.Bake("commit", "-m", "testrepo-a commit 1").RunAndReport()
				dataHash_2 = rio.CommitID(strings.Trim(git.Bake("rev-parse", "HEAD").Output(), "\n"))
				ioutil.WriteFile("file-e", []byte("efghi"), 0644)
				git.Bake("add", ".").RunAndReport()
				git.Bake("commit", "-m", "testrepo-a commit 2").RunAndReport()
				dataHash_3 = rio.CommitID(strings.Trim(git.Bake("rev-parse", "HEAD").Output(), "\n"))
				// Create submodule
				git.Bake("init", "--", "subrepo-a").RunAndReport()
				testutil.UsingDir("subrepo-a", func() {
					ioutil.WriteFile("./submodule-file-1", []byte("zxcvb"), 0644)
					// Create sub-submodule
					git.Bake("init", "--", "subrepo-b").RunAndReport()
					testutil.UsingDir("subrepo-b", func() {
						ioutil.WriteFile("./submodule-file-2", []byte("fihasd"), 0644)
						git.Bake("commit", "-m", "sub sub repo commit").RunAndReport()
					})
					git.Bake("submodule", "add", "--", "./subrepo-b").RunAndReport()
					git.Bake("commit", "-m", "sub repo commit").RunAndReport()
				})
				git.Bake("submodule", "add", "--", "./subrepo-a").RunAndReport()
				git.Bake("commit", "-m", "testrepo-a commit add submodules").RunAndReport()
				dataHash_4 = rio.CommitID(strings.Trim(git.Bake("rev-parse", "HEAD").Output(), "\n"))
			})

			transmat := New("./workdir")

			Convey("Materialization should be able to produce the latest commit", FailureContinues, func() {
				uris := []rio.SiloURI{rio.SiloURI("./repo-a")}
				// materialize from the ID returned by foreign git
				arena := transmat.Materialize(Kind, dataHash_4, uris, testutil.TestLogger(c), rio.AcceptHashMismatch)
				// assert hash match
				// (normally survival would attest this, but we used the `AcceptHashMismatch` to supress panics in the name of letting the test see more after failures.)
				So(arena.Hash(), ShouldEqual, dataHash_4)
				// check filesystem to loosely match the original fixture
				So(filepath.Join(arena.Path(), "file-a"), testutil.ShouldBeFile)
				So(filepath.Join(arena.Path(), "file-e"), testutil.ShouldBeFile)
				So(filepath.Join(arena.Path(), ".git"), testutil.ShouldBeNotFile)
				// Check submodule exists
				So(filepath.Join(arena.Path(), "subrepo-a", "submodule-file-1"), testutil.ShouldBeFile)
				// Check submodules of submodules do NOT exist
				So(filepath.Join(arena.Path(), "subrepo-a", "subrepo-b", "submodule-file-2"), testutil.ShouldBeNotFile)
			})

			Convey("Materialization should be able to produce older commits", FailureContinues, func() {
				uris := []rio.SiloURI{rio.SiloURI("./repo-a")}
				// materialize from the ID returned by foreign git
				arena := transmat.Materialize(Kind, dataHash_2, uris, testutil.TestLogger(c), rio.AcceptHashMismatch)
				// assert hash match
				// (normally survival would attest this, but we used the `AcceptHashMismatch` to supress panics in the name of letting the test see more after failures.)
				So(arena.Hash(), ShouldEqual, dataHash_2)
				// check filesystem to loosely match the original fixture
				So(filepath.Join(arena.Path(), "file-a"), testutil.ShouldBeFile)
				So(filepath.Join(arena.Path(), "file-e"), testutil.ShouldBeNotFile)
				So(filepath.Join(arena.Path(), ".git"), testutil.ShouldBeNotFile)
				// Check submodule does not exist
				So(filepath.Join(arena.Path(), "subrepo-a", "submodule-file-1"), testutil.ShouldBeNotFile)
				// Check submodules of submodules do NOT exist
				So(filepath.Join(arena.Path(), "subrepo-a", "subrepo-b", "submodule-file-2"), testutil.ShouldBeNotFile)
			})

			Convey("Materialization should work, even when cwd is inside the repo", FailureContinues, func() {
				So(os.Mkdir("repo-a/meta", 0755), ShouldBeNil)
				testutil.UsingDir("repo-a/meta", func() {
					uris := []rio.SiloURI{rio.SiloURI("./..")}
					// materialize from the ID returned by foreign git
					arena := transmat.Materialize(Kind, dataHash_3, uris, testutil.TestLogger(c), rio.AcceptHashMismatch)
					// assert hash match
					// (normally survival would attest this, but we used the `AcceptHashMismatch` to supress panics in the name of letting the test see more after failures.)
					So(arena.Hash(), ShouldEqual, dataHash_3)
					// check filesystem to loosely match the original fixture
					So(filepath.Join(arena.Path(), "file-a"), testutil.ShouldBeFile)
					So(filepath.Join(arena.Path(), "file-e"), testutil.ShouldBeFile)
					So(filepath.Join(arena.Path(), ".git"), testutil.ShouldBeNotFile)
					// Check submodule does not exist
					So(filepath.Join(arena.Path(), "subrepo-a", "submodule-file-1"), testutil.ShouldBeNotFile)
					// Check submodules of submodules do NOT exist
					So(filepath.Join(arena.Path(), "subrepo-a", "subrepo-b", "submodule-file-2"), testutil.ShouldBeNotFile)
				})
			})
		})),
	)

	// TODO you really should do this with a fixture loop
	// but that does also leave questions about multi-commits, branches, etc.
	// so do both i guess.
	//filefixture.Beta.Create("repo-a")
}