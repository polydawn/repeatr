Repeatr Code Layout Overview
----------------------------

:warning: This document is intended for developers and contributors to the Repeatr core and plugins.
If you're using Repeatr, but not interested in modifying it, you can skip this doc.

- `api`
  - `api/def`: all the specs of types within the system.
    - so many things -- pretty much the entire set of nouns in the big-picture docs:
      - Wares, Warehouses
      - Formulas, Actions, MountOpts, Filters
      - RunRecords, Catalogs, Commissions
	- no external imports -- easy to link if another program wants to speak our API.
  - `api/act`: API proxying functions and other code-in-motion.
    - compared to `api/def`, functions under this package do work; `def` is purely API types.
	- no external imports -- easy to link if another program wants to speak our API.
  - `api/hitch`: leftover bits (e.g. yaml parsing).  Has messier imports.
- `core`
  - `core/executors`: containment engines, and all the stuff to execute a formula.
    - `core/executors/cradle`: Features for minimum-viable environment setup.
    - `core/executors/impl/*`: Plugins (chroot, runc, etc) are clustered under this tree.
	- `core/executor/tests`: Specs for behaviors all executors should match.
  - `core/actors`: shared code components which require their own threads/goroutines to run, performing nontrivial bits of orchestration between concurrent components.
  - `core/assets`: helper package for managing internal references to assets not compiled into the Repeatr binary itself.
  - `core/jank`: the inevitable junk drawer -- lost code and overly global references that need refactor.
- `rio`
  - The heavy-lifting parts of data transport and Repeatable Input/Output.
    - `rio/assets`: Mechanisms for bootstrapping other plugins that have their own file bundles.
    - `rio/filter`: Filters to run on filesystems (usually to strip nondeterministic bits).
    - `rio/placer`: Plugins for reshuffling filesystems cheaply (bind mounts, COW tools, etc).
    - `rio/transmat`: All of the actual heavy lifter code for moving bytes!
      - `rio/transmat/impl/*`: Plugins (tar, s3, git, etc) are clustered under this tree.
	- `rio/tests`: Test specs which all transmat implementations must comply with.
  - May depend on `lib`, but that's it -- a command could be built on `rio` and do all repeatr's data transport stuff
  - All components are usable *without* any of the rest of repeatr's container features.
  - Hypothetically should work just fine cross-platform (but don't assume it unless you see test coverage).
- `lib`
  - Misc library bits.
  - Any things that could easily be broken out into externalized libraries entirely, but aren't basically because they aren't significant enough to justify the management overhead of their own git repos.
  - Does *not* import from *any* other trees of the codebase.  "Could easy be broken out".
  - Some IO stuff (and even hashing of it) falls into `lib` on the theory it might be reusable for a large-data-diffing tool, or similar.
- `doc`
  - You're lookin' at it.
- `meta`
  - Build utilities, Release scripts, and other meta-project leftovers.
- `cmd`
  - The comand-line interface and start of the program are here.
- `rsrch`
  - Code here is not ready for prime-time and not referenced by any other packages, but contains draft structures that may become part of future iterations of Repeatr.
