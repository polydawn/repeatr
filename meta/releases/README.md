Repeatr releases are scripted in repeatr; and they are, of course, expected to be
deterministic.

This dir contains all the formulas, and a script to run the entire set of them.
`go run ./meta/releases/main.go` from the git repo root is the normal usage.

The binaries are emitted in CAS layout.
Symlinks point to them by name for the convenience of the downloads links in the website
(and these are also the same filenames as we provide from github's releases section).
Symlinks are committed to git at the point we designate a new release.
None of the binaries are committed to git, but since they should all be regenerateable,
it's actually valid for us to assert no symlink should be dangling after you run all
builds (so, we do; this currently functions as our is-it-stable-over-time check).

The script that executes all formulas, will also handle the results,
either checking the results against the existing references kept in symlinks,
or creating the named links (the maintainer will commit these when releasing).

Formulas explicitly specify GOOS and GOARCH for clarity (at present this is redundant,
because the pinned go compiler has these defaults as well).

Formulas prefer to be run from this cwd.
This isn't a strict requirement, but relative paths to assets are set up to e.g.
refer to your local git clone for convenience, etc, if this cwd is respected.

Formulas are quite redundant between each version;
the build & packaging process typically does not change significantly from release to release.
We keep each of them regardless, in keeping with the principle that they should be immutable.
Much of this which seems like ad-hoc use of git and symlinks... *is*:
these systems are a stand-in expected to be replaced with more reusuable,
friendly tools as the pipelining and permanent record keeping models further mature.
