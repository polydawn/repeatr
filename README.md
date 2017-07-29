# Repeatr [![Build status](https://img.shields.io/travis/polydawn/repeatr/master.svg?style=flat-square)](https://travis-ci.org/polydawn/repeatr) [![GoDoc](https://godoc.org/github.com/polydawn/repeatr?status.svg)](https://godoc.org/github.com/polydawn/repeatr)

```
repeatr run same_thing_in > same_thing_out
```

Repeatr is a tool for running processes repeatedly.  Repeatr is designed to make task definition precise, environment setup portable, and results reproducible.

Some of Repeatr's key features and goals include:

- *Zero-ambiguity environment*: Repeatr is developed on the principle of "precise-by-default".  All files in your environment are managed by content-addressible storage (think: pinned as if by a git commit hash).
- *Deep-time reproducibility*: Repeatr represents a commitment to reproducible results today, tomorrow, next week, next year, and... you get the picture.  Repeatr configuration explicitly enforces a split between << data identity >> and << data location >>.  The former never changes; the latter is explicitly variable.
- *Communicable results*: Repeatr describes processes in a [Formula](doc/formulas.md).  Communicating a Formula -- via email, gist, pastebin, whatever -- should be enough for anyone to repeat your work.
- *Control over data flow*: Pull input files from multiple systems; explicitly declare sections of filesystem that are useful results to pass along.  Granular control lets you build pipelines that are clean, explicit, and fast.
- *Variation builds on precision*: Repeatr designs for systems like automatic updates and matrix tests on environmental variations by building them *on top* of Formulas.  This allows clear identification of each version/test/etc, making it possible to clearly report what's been covered and what needs to be finished.  Other tools can generate and consume Formulas as an API, plotting complex pipelines and checking reproducibility of results however they see fit.

Repeatr is *not* a build tool; think of it more as a workspace manager.
It's important to have a clean workspace, fill it with good tools, and keep the materials going both in and out of your workspace well-inventoried.
You can use `make`, `cake`, `rake`, `bake`, or whatever's popular this month inside Repeatr; Repeatr gives you a framework to make sure everyone plays nice.



Why?
----

Building software should be like `1 + 2 = 3` -- start with the same numbers,
add 'em, and you should always get the same thing.

Repeatr is applying the same theology to complex systems -- write down all the
inputs precisely, do the same operation, and always get the same thing.
(Yes: it *IS* possible.)

Repeatr's first and foremost goal is to make sure your process is *repeatable*:
that we can run it again (whether it produces the same output or not is your responsibility).
Repeatr's second goal is to make it very easy to check if your process is *reproducible*:
if we get the same thing out when we put the same things in.

Repeatr can't magically make your project reproducible.  But it can help!
By introducing [Formulas](doc/formulas.md) as a concrete language to define repeatable execution,
and providing a standardized way to define outputs which includes fast, efficient checking for identical results,
reproducibility because easy to measure, and thus easier to achieve.



Okay, but why should *I* care?
----------------------------

Work *better* by using reproducible processes to remove variables.
Getting the ability to repeat a process locked down early makes working with it safer and easier,
and moving forward *simpler*.

Compared to other container systems,
Repeatr's abilities to synthesize filesystems from multiple sources,
pin clear versions all the moving parts,
and export only the results you want --
all these features work together to make working with complex systems *faster*,
and easier to split into pipelines with clear interactions and predictable results.



Read more
---------

- [Why is Repeatability Important?](doc/why-repeat.md)
- [Formulas](doc/formulas.md)
- [Containers & Execution](doc/containers.md)
- [Pipelines & Iteration](doc/continuity.md)
- [Glossary](doc/glossary.md)
- Developer docs:
  - [Building Repeatr](doc/dev/building-repeatr.md)
  - [Code Layout](doc/dev/code-layout.md)



Project Status
--------------

Alpha.  Feel free to use it in whatever environment you like, but currently,
breaking changes to formats are possible and there's not going to be a lot of
hand-holding on migrations until we reach a higher level of maturity.

That said, Repeatr is self-hosting Repeatr's builds, so, we're not entirely
*un*-invested in stability, either :)

Working:

- Filesystem transport pulling in from git, and storing content in tar (with plugins for working directly with http, AWS S3, and GCS).
- [Contained execution](doc/containers.md) using pluggable backends for isolation.
- [Formulas](doc/formulas.md) as a format for declaring filesystems, environments, a process, and what outputs to scan and keep.
- Results returned as a Formula including content-addressible hashes over the requests outputs.
- Configurable filters to automatically remove the most common problems with reproducibility (file timestamps, local permissions, etc).
- Automatically uploading result filesystems.  Keep 'em as records, or use them in the next stage of a multi-step process.
- Easily drop privileges inside the container.  The *defaults* give you a non-zero UID and strictly limited capabilities.  Set a single string parameter to switch to another coherent security policy.

Future work:

- More filesystem transport plugins.  (This todo will be eternal; there's always something more we could integrate with.)
- More executor systems (currently all our choices are linux containers at heart; full VMs would be nice.)
- More robust error handling and user-facing messaging.
- Better tooling around mirroring storage, and cleaning up old/dangling/uninteresting objects.
- See also the [roadmap](ROADMAP.md).



Getting Started
---------------

First, [get Repeatr](http://repeatr.io/install).
Or, if you'd prefer to build from source, follow the [build instructions](doc/dev/building-repeatr.md).

Then, try out the demo script in this repo: `demo.sh` covers a bunch of basic functions
and has formulas you can fork to start your own projects.

Or, look at how Repeatr builds Repeatr repeatedly: `repeat-thyself.sh` is a real-world example
of a full software build -- Repeatr's.

When in doubt, try `repeatr help` or `repeatr [subcommand] help`.
These should give you more information about what the various commands do and how to use them.



Contributing
------------

- Repeatr is Apache v2 licensed.  We're very gung ho on freedom, and if you'd like to help, the more the merrier!
- First time building?  [We've got a doc for that](doc/dev/building-repeatr.md).
- Need help navigating the code?  Check out the [code layout overview](doc/dev/code-layout.md).
- Ready to propose a code change?  Kindly give the [contribution guidelines](CONTRIBUTING.md) a gander.
- Just have a spelling correction or grammar nit?  Every little bit helps, please send 'em in!



Errata
------

Repeatr tries to use the most efficient systems available on your host by default.
Specifically, for making copy-on-write filesystems for isolating jobs, if you have AUFS available,
repeatr will use it; if you don't, repeatr falls back to doing a (much slower) regular filesystem copy,
and warn you that it's taking a slow route.
How you install AUFS may very per system, but on ubuntu `apt-get install aufs-tools` should work.
