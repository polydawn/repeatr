Repeatr Code Layout Overview
----------------------------

:warning: This document is intended for developers and contributors to the Repeatr core and plugins.
If you're using Repeatr, but not interested in modifying it, you can skip this doc.


- `cli`, `cmd`
  - The comand-line interface and start of the program are here.
- `def`
  - The core model definitions of Repeatr -- formulas, wares, etc.
  - No logic here (and no external dependencies!  This is meant to be importable as a library without a fuss if you want to work with Repeatr's data model).
- `executor`
  - Containment engines and their tests.
  - Plugins (chroot, runc, etc) are clustered under this tree.
- `io`
  - The data transport components of Repeatr.
  - No references to execution, etc: Meant to be reusable outside of Repeatr.  (Want to write a backup client on the same integrity-guaranteed storage libraries?  Do eet!)
  - `io` has a lot of stuff going on inside:
    - `io/assets`: Mechanisms for bootstrapping other plugins that have their own file bundles.
    - `io/filter`: Filters to run on filesystems (usually to strip nondeterministic bits).
    - `io/placer`: Plugins for reshuffling filesystems cheaply (bind mounts, COW tools, etc).
    - `io/transmat`: All of the actual heavy lifter code for moving bytes!
      - Plugins (tar, s3, git, etc) are clustered under this tree.
- `lib`
  - Misc library bits.
- `scheduler` -- deprecated, forget it exists.
- `testutil`
  - Code only meant to be used in testing.  If this is ever referenced by another package outside of the "_test.go" files, it's a bug.
- actors -- coming soon
- catalog -- coming soon
- model -- coming soon
