# repeatr [![Build status](https://img.shields.io/travis/polydawn/repeatr/master.svg?style=flat-square)](https://travis-ci.org/polydawn/repeatr)

```
repeatr run same_thing_in > same_thing_out
```

Clean sandboxes, made with free-range locally-sourced perfectly manicured sand.

Half joking, also for reals: repeatr gives you sandboxes where there's an audit trail for every grain of sand in your environment, detailing where it came from and how it was formed.



Technical Elevator Pitch
------------------------

- Containers!
- Content-addressable storage!
- Decentralized! Host your own!

Repeatr combines sandboxing (so your know your application is working independently from the rest of the system)
with data provisioning based on immutable IDs that pinpoint exactly one thing (meaning you can always start with a known system state).
Matching these two properties means we can make things that are repeatable,
reliably working the same way, month after month... even year after year, and decade after... well, you get the idea.

What's more: anything you run in repeatr can spit out more data and we'll give it the same kind of immutable IDs.
These can be used to build the environment for another process: [formulas](doc/formulas.md) have the same inputs as outputs.
Chaining processes together is easy.
Building an entire system this way means every step is auditable and reproducible by anyone with access to the raw materials.

Repeatability is the cornerstone of science and engineering.
`repeatr` is about making it easy for all your digital stuff.



What is this good for?
----------------------

Data processing.  Science.  Building software.  Doing analysis.  Responsible journalism.
Anywhere where it's appropriate to show your work, it's appropriate to use Repeatr.

Consider the following:

- Sandboxing to make sure your processes are consistent -- like a Continuous Integration system, but also ready to help you debug consistency itself.
- Immutable deployments -- and we mean it.  Once the formula is committed, everything is pinned down and guaranteed to deliver.  Ops teams delivering IT infrastructure can rest easy.
- Data warehousing -- where file corruption can mean millions of dollars in damage from inaccurate data or expensive calculations that need to be re-run, repeatr detects issues before they cause problems.  Storing data on untrusted third-party storage is safe because integrity guarantees are baked in to the system.
- Repeatable, reproducible pipelines -- when transparency is important, repeatr makes planning work and leaving an audit log one and the same: exact reproducibility is simply natural.
- Roll it back -- formulas used in the past continue to work.  Forever.  If you change your data, and later discover you want to run with an older configuration again, that's *always* possible.

If you're a programmer, think of it like source control, but for your entire environment, with precise commits, rollback, and even bisect capabilities.

If you're a researcher, think of it like your lab notebook, but you write the notebook first, and then Repeatr runs the experiment *for* you according to your exact instructions.

If you're a journalist, think of it like citing sources, but not only can readers and editors look up your citations, they can also run the analysis themselves in one click.

In any situation where quality is critical and transparency is a must, repeatr can help you raise the bar.



Where's the Magic?
------------------

For more about what makes Repeatr special, check out these docs:

- [doc/formulas](doc/formulas.md) : A Formula is how Repeatr describes a precise unit of work.
- [doc/containers](doc/containers.md) : Repeatr is container-agonostic, and supports several choices for how to get the isolation you need.



Get Started Repeating
---------------------

First, [get Repeatr](http://repeatr.io/install).
Or, if you'd prefer to build from source, follow the [dev build docs](doc/dev/building-repeatr.md).

Then, try out the demo script in this repo: `demo.sh` covers a bunch of basic functions
and has formulas you can fork to start your own projects.

Or, look at how Repeatr builds Repeatr repeatedly: `repeat-theyself.sh` is a real-world example
of a full software build -- Repeatr's.

When in doubt, try `repeatr help` or `repeatr [subcommand] help`.
These should give you more information about what the various commands do and how to use them.



Contributing
------------

- Repeatr is Apache v2 licensed.  We're very gung ho on freedom, and if you'd like to help, the more the merrier!
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
