issues
======

showstoppers
------------

- can't slurp data from a local file repo at all
  - it's partially there: `storage/seekable` is for local file repos
  - blocker 1: doesn't support reading loose data objects on local (only packfiles)
    - look at the `(s *ObjectStorage) Get` method.  only packfiles.
    - `formats/objfile` looks like looses thought, both read and write
      - so maybe the amount of glue missing is quite tiny at this point?
        - indeeeeed.  `storage/seekable` straight up just doesn't import the objfile package yet.
        - looks like most of the objfile support landed on july 4 (not too long ago).
  - blocker 2: packfile support is not completely.  only works for *single* packfiles.
    - look at `(d *GitDir) Packfile()` method.  the very name should be plural.  :/
    - this is probably fairly easily fixed.

- can't produce a clone that cgit can read
  - note: lower priority (we don't currently, in fact, do this)
  - blocker 1 alt1: can't write packfiles
  - blocker 1 alt2: maybe could cludge it with objfiles alone?  but also requires writing.
  - blocker 2: not sure if it can write the rest of the amusing config files at all

- the """// TODO: Provide "haves" """ thing is kind of a perf issue!
  - i actually see the haves protocol bits done `clients/common`
    - http uses em.  ssh doesn't appear to.


unknown
-------

- submodule support?
  - a PITA to write if not already present, but i experienced this once before in jgit -- know roughly what to do.


absent, but irrelevant
----------------------

- git protocol support
  - would be nice to have, but use in the wild is very very rare

- ability to push
  - on any protocol.  neither http or ssh support it.
  - we basically don't do this and don't expect to.


probably good
-------------

- ssh support
  - may need to write our own config parse? :/
  - otherwise, looks good, might actually be easier to debug and maintain that host stuff
    - and let's not even talk about how much better this is than the linking issues with libgit2
  - didn't see where the known_hosts can be specified yet but i'm sure the "x/crypto/ssh" pkg did something sane.
- http support
  - not much to say.  we've already seen it working.
  - only supports the http smart transport, not the legacy dumb transport.
    - the dumb http system isn't used much in the wild.  except i certainly do sometimes, tbh.  so easy to serve.
	- if we do start hacking on this, worth noting that the dumb http transport means in-git-data-dir paths are not actually only an implementation detail of a local fs repo, surprisingly.

- replacing lsremote is definitely easy
  - `NewAuthenticatedRemote(url, auth).Connect().Info().Refs` bam done
  - again, let's not even talk about how much time i spend trying to do that in libgit2 before conclusively discovering it couldn't be done, and my PR to begin to joust at poking that is still in limbo for want of tests around nulls for an api that was already swiss cheese ok breathe take deep breaths it's ok the c isn't here anymore
