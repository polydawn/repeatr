Developing Repeatr
==================

:warning: This document is intended for developers and contributors to the Repeatr core and plugins.
If you're using Repeatr, but not interested in modifying it, you can skip this doc -- feel
free to download a binary release from [repeatr.io](http://repeatr.io/install#downloading-nightlies).



Building Repeatr
----------------

Building repeatr from source requires a working bash shell environment, a golang compiler, and git (to fetch the repeatr source code).

If you already have git and a golang compiler installed, you can skip a lot of steps; otherwise, this bash snippet should put everything together for you (on debian-family linuxes; other distros and mac users, substitute your favorite system package managers):

```bash
### install git, if you don't have it already
sudo apt-get install git
### install golang
wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
tar -xf go1.8.linux-amd64.tar.gz
export GOROOT=$PWD/go
export PATH=$PATH:$GOROOT/bin
### clone and build repeatr
git clone https://github.com/polydawn/repeatr.git
cd repeatr
./goad init
./goad install
```

This should leave a `repeatr` executable at `.gopath/bin/repeatr`.

This executable is statically linked[1] and all you need to start running with repeatr.

You can also use `./goad sys` to install the repeatr binary to a systemwide path
(it simply copies the executable to `/usr/bin/repeatr` for your convenience).

[1] Almost.  Our build does still link libc by default, and uses it for network name resolution services.



Testing Repeatr
---------------

We have three categories of test in Repeatr:

- Unit tests -- `./goad test` will run these; they make up the bulk of the testing.
- Acceptance tests -- `./goad test-acceptance` will run these; you'll need to run `./lib/integration/assets.sh` once first to get some of the bulkier contents in place.
- Demo (semi-interactive) -- `./demo.sh` runs this, though it's also covered by the acceptance tests.

You can also run `./goad bench` to run performance benchmarks for the components that have them.


Note that several sections of repeatr require elevated privileges to run
(sandboxing and mounting, ironically, require high priviledges).
The test suite for these areas will be skipped without root!
To run the *whole* suite, make sure to start the tests as root, and
check for any unexpected skipped entries in the test report.



Repeating Repeatr
-----------------

Repeatr can be used to build Repeatr repeatedly and reproducibly!
Check out the [repeat-thyself](../../repeat-thyself.sh) script in the source repo.
It contains a repeatr formula, and will automatically extract the current git commit
hash you have checked out, and insert that into the formula to produce an isolated
and completely reproducible build.

This is an excellent end-to-end test, not only because self-hosting tools are cool,
but because Repeatr *is* reproducible -- if you press up-enter up-enter up-enter
at your terminal, you should get the same results over and over :tada:

Binary Repeatr releases are built using formulas which we've committed in the `meta/releases/` dir.
There is a script at `meta/releases/main.go` which will execute all of them in bulk,
but each individual formula should run fine in a regular `repeatr run whichever.formula`.

You're welcome to check that you can reproduce our binaries!  Please do!
In an ideal world, you should have absolutely no requirement to trust the published Repeatr releases.
Not saying that we're bad people -- just that "trust, but verify" is a winning strategy.
Any Repeatr binary built from source should be able to reproduce the others;
and any Repeatr binary should be able to reproduce the others.
Variations in the version of Go compiler will of course produce different binaries,
but should also produce functionally interchangeable results for additional dimensions of verification.
Experiment to your heart's content!
