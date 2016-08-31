issues
======

- the """// TODO: Provide "haves" """ thing is kind of a perf issue!
- doesn't support reading loose data objects on local (only packfiles)
  - wait, are their docs out of date?  `storage/seekable` sounds like looses
    - nope, it's not looses.  look at the `(s *ObjectStorage) Get` method.  only packfiles.
	  - and only *single* packfiles at that.
	    - `(d *GitDir) Packfile()` demonstrates a fundamental misunderstanding.  that should be plural.  :/
    - `formats/objfile` looks like looses thought, both read and write
	  - so maybe the amount of glue missing is quite tiny at this point?
	    - indeeeeed.  `storage/seekable` straight up just doesn't import the objfile package yet.
		- looks like most of the objfile support landed on july 4 (not too long ago).