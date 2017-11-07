Repeatr
=======

```
repeatr run same_thing_in > same_thing_out
```

Repeatr is a tool for running processes repeatedly.  Repeatr is designed to make task definition precise, environment setup portable, and results reproducible.

Some of Repeatr's key features and goals include:

- *Zero-ambiguity environment*: Repeatr is developed on the principle of "precise-by-default".  All files in your environment are managed by content-addressible storage (think: pinned as if by a git commit hash).
- *Deep-time reproducibility*: Repeatr represents a commitment to reproducible results today, tomorrow, next week, next year, and... you get the picture.  Repeatr configuration explicitly enforces a split between << data identity >> and << data location >>.  The former never changes; the latter is explicitly variable.
- *Communicable results*: Repeatr describes processes in a [Formula](https://polydawn.github.io/glossary#formula).  Communicating a Formula -- via email, gist, pastebin, whatever -- should be enough for anyone to repeat your work.
- *Control over data flow*: Pull input files from multiple systems; explicitly declare sections of filesystem that are useful results to pass along.  Granular control lets you build pipelines that are clean, explicit, and fast.
- *Variation builds on precision*: Repeatr designs for systems like automatic updates and matrix tests on environmental variations by building them *on top* of Formulas.  This allows clear identification of each version/test/etc, making it possible to clearly report what's been covered and what needs to be finished.  Other tools can generate and consume Formulas as an API, plotting complex pipelines and checking reproducibility of results however they see fit.

Repeatr is *not* a build tool; think of it more as a workspace manager.
It's important to have a clean workspace, fill it with good tools, and keep the materials going both in and out of your workspace well-inventoried.
You can use `make`, `cake`, `rake`, `bake`, or whatever's popular this month inside Repeatr; Repeatr gives you a framework to make sure everyone plays nice.


More documentation
------------------

Repeatr is just one part of an ecosystem of software called the Timeless Stack.

The Timeless Stack documentation has its own repo: https://github.com/polydawn/timeless

Much of the documentation is published in html book form: https://polydawn.github.io/



:warning: Alpha Warning :warning:
---------------------------------

Repeatr is in Alpha.

This branch is the "v0.200" development branch -- it's all the accumulated API changes we've learned we wanted to make after several years.
Some docs may refer to the older versions; please forgive the mess as we transition.

This "v0.200" branch is nearly up to feature-parity with the previous versions already; we recommend being up here on the new stuff if you can,
but the "v0.15" releases are still stable and available if you so choose.

Despite being in alpha, we consider the repeatr API fairly stable, and are happy to recommend building with it.
We made one set of breaking changes out of the last 2.5 years; and we expect this API to last at least twice as long as the previous one.

Binary releases releases are available from the [github releases page](https://github.com/polydawn/repeatr/releases).
It's also easy to build from source if you want the absolute latest bleeding-edge features.



Building from Source
--------------------

Git-clone, then in the repo dir:

```
fling init          # fetch libraries
fling install-deps  # build rio component (used to fetch other plugins)
fling fetch-plugins # fetch plugins
fling               # build & test
```

Future incremental builds are just `fling` -- the rest of that was all first-time setup.

Binaries go into the `bin/` dir; add it to your $PATH.

Libraries are handled via git submodules.  You can run `fling init` again at any time to re-sync them.
Plugins are handled via `rio`, another part of the Timeless Stack that Repeatr builds upon.

You can use `fling -h` to see other individual build and test command options.
For example `fling test` will only run tests; `fling install` will not test, just build binares in `bin/`.
