# repeatr [![Build status](https://img.shields.io/travis/polydawn/repeatr/master.svg?style=flat-square)](https://travis-ci.org/polydawn/repeatr)

```
repeatr run same_thing_in > same_thing_out
```

Clean sandboxes, made with free-range locally-sourced perfectly manicured sand.

Half joking, also for reals: repeatr gives you sandboxes where there's an audit trail for every grain of sand in your environment, detailing where it came from and how it was formed.





Why?
----

Repeatability is the cornerstone of science and engineering.

(No?  Okay, go read [doc/why-repeat](doc/why-repeat.md) and I'll see if I can convince you!)



How?
----

- Containers!
- Content-addressable storage!
- A Formula!

Repeatr combines sandboxing (so your know your application is working independently from the rest of the system)
with data provisioning based on immutable IDs that pinpoint exactly one thing (meaning you can always start with a known system state).
Matching these two properties means we can make things that are repeatable,
reliably working the same way, month after month... even year after year, and decade after... well, you get the idea.

What's more: anything you run in repeatr can spit out more data and we'll give it the same kind of immutable IDs.
These can be used to build the environment for another process: [formulas](doc/formulas.md) have the same inputs as outputs.
Chaining processes together is easy.
Building an entire system this way means every step is auditable and reproducible by anyone with access to the raw materials.


Where's the Magic?
------------------

For more about what makes Repeatr special, check out these docs:

- [doc/formulas](doc/formulas.md) : A Formula is how Repeatr describes a precise unit of work.  The Formula is a language-agonostic API suitable for describing any environment.
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
