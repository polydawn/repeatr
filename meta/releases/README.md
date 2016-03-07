Releases
========

Getting them
------------

Binary release tarballs are available from GitHub: https://github.com/polydawn/repeatr/releases/
(Individual downloads have URLs like https://github.com/polydawn/repeatr/releases/download/release%2Fv0.12/repeatr-linux-amd64-v0.12.tar.gz ).

You may be thinking to yourself "dang, bootstrapping is hard" and
"how do I get this awesome tool for fetching and installing pinned, reliable versions of stuff... in an awesome way that's also pinned?"
Try (toolstrap.sh)[toolstrap.sh] as a working example of how to grab a Repeatr release and install it using just bash and system tools like `sha384sum`.
Or, use whatever works for you!  This script is simply a readymade option with minimal deps -- it's not the only answer, just a solid answer if you're looking for a starting point.


Building them
-------------

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
You should be able to run them anywhere, but if you start with some other working dir,
you make need to create directories for assets to be stored.
If you use this cwd, those directories are already there for you, and the formulas will
refer to your local git clone for convenience, etc.

Formulas are quite redundant between each version;
the build & packaging process typically does not change significantly from release to release.
We keep each of them regardless, in keeping with the principle that they should be immutable.
Much of this which seems like ad-hoc use of git and symlinks... *is*:
these systems are a stand-in expected to be replaced with more reusuable,
friendly tools as the pipelining and permanent record keeping models further mature.
