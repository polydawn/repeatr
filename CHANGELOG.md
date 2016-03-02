recent /// not yet released
---------------------------

- Bugfix: When using 'tar' transports with 'http' or 'https' URLs, HTTP status codes of 404 will now be reported as `DataDNE` errors.  Previously, this would be incorrectly reported as existing but corrupt data.
- Change: `repeatr run` now accepts the formula file as a positional argument (you can get rid of the `-i ` in all your scripts).
- Change: The default executor in `repeatr run` is now 'runc' instead of 'chroot'.  (You can continue to use the chroot executor by flagging `-e chroot`.)
- Improvement: All transports which store filesystems in tar format (this includes 'tar', 's3' and 'gc') now upload gzip compressed data by default.


v0.11 -- f6095637190f63c9318df74a93c51a230ff85176 -- 21 Feb 2016
----------------------------------------------------------------

This release of Repeatr includes the "Policy" system -- this is majorly exciting: for the first time, we have containers which drastically reduce the privilege of processes inside them by default.
This is a major improvement to security for users, and hopefully the start of major improvements to the whole ecosystem, since safe operations are now the default operations.
Of course, it's also a massively breaking change for any formulas that previously required powerful and unsafe system permissions -- they now have to admit it up-front!  ;)

- Feature: Policies!!  And graceful de-escalation of privileges.  [PR: [gh#68](https://github.com/polydawn/repeatr/pull/68)]
  - By default, executors will drop to user-level privileges and a non-0 (a.k.a non-root) UID.
  - Executors which support advanced features like [linux capabilities](http://man7.org/linux/man-pages/man7/capabilities.7.html) will also drop those.
  - Policy levels available are, from safest to most empowered: `routine`, `uidzero`, `governor`, and `sysad`.  Routine is the default.
- Feature: Several minimum-viable-provisioning will be applied to your filesystems and environment before job launch: this is called the "cradle".  These features make operating with low privileges (as introduced concurrently by the policies feature) much easier.
  - If you configure a `cwd` that doesn't already exist, it will be automatically created and be writable.
  - Your jobs may now reliably expect `/tmp` to exist and to be writable (specifically, it will be forced to chmod=01777; world-writable plus sticky bit, as a tempdir should be).
  - The `$HOME` environment variable will now be assigned by default.  The referenced directory will exist (and be writable, if cradle created it).
  - These new behaviors can be disabled by configuring `action.cradle = false` in your formulas.
- Bugfix: Clean up the filesystem more gingerly if major errors are raised during executor operation.  Certain failure cases of unmounting could previously cause more files to be removed during "cleanup" -- if you're using host mounts, this could be a fairly major problem and you should upgrade immediately.
- Bugfix: Files produced by the 'git' transport will now by owned by uid=1000, gid=1000.  This is consistent with the default filter values for other transports.
- Internal: Defining a mechanism to feed results of one formula into another, describing ways to communicate well-known ware hashes by name, and thereupon build automatic update systems and complex processing pipelines.  Proof-of-concept work -- will not be externally exposed or API-stable for some time.  [PR: [gh#67](https://github.com/polydawn/repeatr/pull/67)]


v0.10 -- 5db67459729206cf6655f59c4acf340e6a4be207 -- 03 Feb 2016
----------------------------------------------------------------

- Feature: The `repeatr twerk` subcommand will now mount your current host directory into the container.  Writably.  Remember: this feature is for exploration and play; it is not the paragon of safe defaults.
- Bugfix: The `repeatr twerk` subcommand now exits non-zero/failure if the contained process itself exits non-zero/failure.  This is consistent with the behavior of `repeatr run`.
- Bugfix: Try harder to enable AUFS.  Previously, Repeatr would sometimes not use the AUFS features even if they were installed.  (Specifically, Repeatr will now attempt to load the kernel module if it's not loaded.  This behavior actually makes me somewhat nervous (it would be preferable to avoid "magicking" a disabled service to life, in general principle, in case it was intentional), but in practice, my own servers seem to regularly have this module unloaded, and it's certainly not something I did on purpose, so it seems likely that this is the practical right thing to do for users.)


v0.9 -- 57183bcb32d7e5fd1aaa4a1c6fe1fd73f877d8d8 -- 26 Dec 2015
---------------------------------------------------------------

- Feature: New transport!  Google cloud storage is now available for inputs and outputs!  Like the 's3' system, it continues to map filesystems to simple and reliable tar formats, and may be used content-addressably (and it shares the same hashing namespace as 's3', 'tar', and 'dir').  Either tokens or service accounts may be used for authorization.
- Changed: The 's3' transmat now specifies content-addressable storage layout with a "s3+ca://" url -- consistent with 'tar' and 'dir'.  The previous format, "s3+splay://", continues to be supported, but is considered deprecated.
- Changed: `repeatr twerk` now uses the 'runc' executor by default!  This should be a mostly silent change thanks to our compatibility specs for executors... except use of a `tty` is also now the default, which should result in a much nicer experience with interactive commands.
- Bugfix: Support more than one use of the 'runc' executor at a time.
- Changed: `repeatr run` now outputs the entire formula, not just the outputs.  This may be a breaking API change for any scripts consuming this output.
- Internal: Updates to formula codec implementation which enable deterministic serialization (namely, outputting maps in a predictable order).


v0.8 -- c408cb473b655fd3d4ba7b7864beac88a038772b -- 29 Nov 2015
---------------------------------------------------------------

- Feature: New subcommand!  `repeatr unpack` can be used to just deliver a filesystem somewhere on a host machine (without any processing).  This is useful if you want to hand files off to some other tooling at the end of a pipeline of Repeatr processes, or if you just want to eyeball 'em to see what happened.
- Feature: New subcommand!  `repeatr explore` can get any ware (same type/hash/silo tuple as everything else) and print out a description of each of its contents.  This is useful for debugging or scripting to get high-level diffs between different snapshots of data.
- Change: The `repeatr scan` tool now applies default filters (stripping uid, gid, and timestamp) matching the defaults for output scanning.  This makes the scan command "Do The Right Thing" in significantly more situations, and is more consistent with the rest of Repeatr.
- Change: The arguments to `repeatr scan` have been renamed; scan, unpack, and explore all line up.
- Change: The 'dir' transport now uses URL formats (e.g. "file://") like the others.  Lack of this was extremely confusing.
- Feature: The 'dir' transport can now use content-addressable storage layout (configured by using "file+ca://"), like the others.
- Improvement: The 'dir' transport now operates transactionally (available in CA mode only).
- Bugfix: Stdout and Stderr from `repeatr twerk` are now correctly proxied.  (Previously the were both commingled to stderr.)  You remain inadvised to use this in any serious scripting, however.


v0.7 -- b15dfa15ffb1f70c3655d06ffaeff8a4a9dd1348 -- 21 Oct 2015
---------------------------------------------------------------

Major shiny: New, ready-to-use container systems.  And host mounts, for when you really want to break out.

- Feature: '[runc](https://github.com/opencontainers/runc)' is now a supported executor!!  It passes all the same compat tests as chroot and nsinit, so they should be interchangeable on the basics, but runc also offers pid namespaces, etc out-of-box, which is a much more excellent experience if your host can support it.  We bundle a link to a specific version of runc for convenience -- no extra setup required.
- Feature: Host mounts!  If your system supports bind mounts (essentially everyone, though sadly not some CI environments), you can now drop writable mounts to your host inside the contained environment.  Doing so of course breaks portability and reproducibility guarantees completely -- but it's dang useful for debug, or simply using repeatr as the reliably shipping system for an intentionally unreproducible user-facing process.
- Improvement: The 'tar' transport will now read a wider range of tars produced by external tools: it will ignore (but warn) on some tar headers which we do not respect, like TypeXGlobalHeader, TypeGNUSparse, etc (this improves compatibility with e.g. source tarballs for gnu projects, which happen to do this a lot).  Our gold standard remains: the hash will have parity with what we *do* to the filesystem, and this maintains our security and precision design guarantees.
- Feature: The 'tar' transport now supports reading xz compression.
- Improvement: Outputs of `repeatr run` are now pretty-printed.


v0.6 -- e9b92540c822a93ed1b8a43b872358160177accd -- 19 Sep 2015
---------------------------------------------------------------

This is a fairly major release.  The format of formulas changed significantly; older formulas are essentially unrecognizable to the tools and will need to be migrated by hand.

- Change: Serial format for formulas changed.
  - You may now use YAML format!  (Multiline strings, rejoice!)  JSON, of course, continues to be acceptable (and recommended, if generating formulas and using them as an API -- json is significantly less ambiguous than YAML).
  - Inputs and outputs are now **maps**.  By default, the map key is interpreted as a mount point (but you may override this, and use meaningful names if you so desire).
  - Note that outputs of `repeatr run` changed to match: outputs are now maps.  This should be a significant ease-of-use improvement: it's now easy to look up an output hash by meaningful name.
  - You may specify a *list* of locations to fetch data from!  Transports that support this will check down the list, giving the ability to failover if one location goes down.
  - Names of keys are now lowercase as per the norm in json.
  - Configuration group `accents` renamed to `actions`.
- Change: `repeatr scan` flag `--uri` renamed to `--silo` (matching the equivalent field name in formula configuration files).
- Feature: Filters can be used to enforce properties on output data.  Examples include filtering UID or GID, and flattening modification timestamps -- these are attributes that are necessary to keep and handle in some situations, but equally often non-semantic and a source of distractions.  So, now you can choose how you want to handle that.  (This has been in the source for a while, but is finally exposed to user config.)
- Feature: Cache filesystems from 'git' transport.  Using the same commit hash in multiple jobs will be instantaneous.  (This does not yet include caching if you fetch different commits from the same repo.)
- Feature: Report DataDNE clearly from 'git' transport.


v0.5 -- 11c6ee9e4daadc29019959d1a7a70de142924744 -- 02 Aug 2015
---------------------------------------------------------------

- Feature: Repeatr self-hosts Repeatr builds!
- Feature: Support for 'git' as a transport!!  This is extremely useful if you want to use repeatr formulas to describe a build and test regimen for a software project.
- Feature: `repeatr run` now exits non-zero/failure if your job itself exits non-zero/failure.  (If you prefer that repeatr only exits with failure if there were problems in the framework around your job, you may specify `--ignore-job-exit`.)
- Feature: `repeatr run` now emits the outputs (json formatted, same schema as the formula, plus hashes) of your job on stdout.  This should be safe to mechanically parse (e.g., `repeatr run | jq [...]`).
- Improvement: Structured logging is now being introduced throughout Repeatr.  Logs now include timestamps, etc, and should be reasonably attractive to look at.


v0.4 -- 013c36f499da68df5d97306cdd4bcd7ab2dcf5a8 -- 01 Aug 2015
---------------------------------------------------------------

- Feature: New subcommand!  `repeatr twerk` will instantly and no-questions-asked create a new sandboxed environment for you.  This is useful for testing and exploration and playtime -- it is *not* recommended for production use, and in particular the contents of the default image are *not* promised to remain consistent over time.
- Improvement: Apply standardized executor behavior tests to nsinit.  Correct several nasty inconsistency with chroot executor.  They should now be reasonably in line with each other.  Testing is good for the soul.
- Feature: Output directories will no longer be automatically created inside the contained environment.  The behavior was problematic in practice; it's better to have jobs explicitly create dirs than implicitly expect them, and extremely silly to have jobs *remove* implicitly created dirs in situations where they are problematic.  This may be a breaking change for your job if it relied on this behavior.
- Feature: The 'tar' transport now accepts tars with implied directories, in order to improve support for some externally produced tars.  The hashing continues to be based on the reality of what's placed on the filesystem, which means while several different tar files might result in the same semantic hash, this does preserve our security and precision design guarantees.


v0.3.1 -- 8248a30e5d8fb78bb243f98534fe652bd2665f61 -- 05 Jul 2015
-----------------------------------------------------------------

- Feature: The 'tar' transport can now fetch from 'https' URLs.
- Feature: Formulas may now specify outputs, but not declare a storage location: this will hash the results, but discard them.
- Feature: Use the fastest filesystem assembly system available to the host system.  Automatically gracefully fall back to slower systems (with a warning) if the optional faster tools aren't available.
- Improvement: Increase amount of isolating applied to chroot (strip env vars from host!).  Standardize testing for executor behaviors.
- Improvement: AUFS mounts are now handled directly by syscall.  Removes need for `mount` command on host system, and fixes leak of mount records.


v0.3 -- 9f4de794fd32cf87cca9109612f0602387150594 -- 15 Jun 2015
---------------------------------------------------------------

With the first appearance of options for fetching input filesystems anonymously from non-local storage, it's now vastly easier to bootstrap a system working with repeatr.

Content-addressable storage modes also put in their first appearance, making it easy to store multiple results in the same storage system.

- Feature: The 'tar' transport can now fetch from 'http' URLs!  This makes formulas much easier to hand off to others.
  - Note: since 'tar' and 's3' already use the same storage format and hashing strategy, this pairs well with using 's3' for upload and storage if you want to share publicly; just set the assets in S3 to public and huzzah, you can have auto-upload and at the same time anyone can use the 'tar' transport to consume the data without mucking about on credentials.
- Feature: The 'tar' transport now supports content-addressable mode!  This means you can store lots of data without bothering to individually name it.  Both local file and http URLs support this.
- Feature: Improve error messaging from data transport.  Separate errors like "can't contact the remote storage" from "the data wasn't there" clearly.


v0.2 -- a82cc8e93e82d37ca798b5c3fdefdeba9e76bd84 -- 01 Jun 2015
---------------------------------------------------------------

- Bugfix: Several breaking changes to the hash modes for 'tar', 'dir', and 's3' transports!
  - Handle hashing of mode bits of files consistently when handling tarballs from other systems.  This does not constitute a security issue (the whitelisting on syscalls was actually *more* strict and consistent than the one on the hashes; security/integrity issues only arise if it goes the other way, where two different data can make it to syscalls but have the same hash), but the fix results in a flag day on hashes.
  - Correctly hash device modes!  Previously, device mode bits would be applied without verification.  This *is* a security issue.  Do not use earlier versions of repeatr.
  - Consistency fixes for "./" prefixes, normalizing treatment of tars from outside of repeatr.
- Feature: New subcommand!  `repeatr scan` will take a local filesystem, hash it, and export it to any one of repeatr's transport plugins for permanent storage.


v0.1.3 -- 281babee641560de89cbb3e5472eea490950b0be -- 24 May 2015
-----------------------------------------------------------------

- Internal: Transport systems now conform to standard interfaces, are more pluggable, etc.
- Feature: Rapid assembly of filesystems from multiple components, using bind mounts, copy-on-write, etc.
- Feature: Caching unpacked filesystems.  Combined with the rapid-assembly features, this means near-instant launch for containers.  Caching is done by hash, meaning cache invalidation Just Works.
- Feature: Locate all caching, job tmpdirs, etc, relative to the `REPEATR_BASE` env var.  This makes it easy to run instances of repeatr with zero overlap (say, so you can stop one, clear its cache, and while the other remains running entirely unphased).  If this var is not set, the default is to share one dir (and thus share caches).
- Feature: Save and fetch filesystem snapshots to and from Amazon S3!  Uses tar format and has the same hash namespace; using the 's3' plugin just instantly takes care of all the details and frees you from single-machine storage.


v0.1.2 -- 7d6ca8c0aa99cd747121d7b0af7548594aefc8eb -- 30 Apr 2015
-----------------------------------------------------------------

- Feature: The 'stdout' and 'stderr' streams from the contained processes is saved to disk, and can be replayed later.
- Feature: `repeatr run` will stream 'stdout' and 'stderr' from the contained process to your terminal in realtime (both will be combined to repeatr's stderr stream).
- Feature: Fetch and unpack filesystem snapshots from tar for use as inputs, using consistent hashing to guarantee integrity and identity.  Works readily with tars produced by other systems, as well.  This means 'tar' finally joins 'dir' in being able to easily provide results from one formula as inputs to another.


v0.1.1 -- 696c00cc0f6db14e01a8c61b9a60ca4415428622 -- 03 Apr 2015
-----------------------------------------------------------------

- Feature: Save filesystem snapshots of results at the end of a job to a directory, using the same deterministic hashing as inputs.  This makes it easy to connect results of one formula to inputs of another formula.
- Feature: Save filesystem snapshots of results via mapping the filesystems into tar format, and hashing deterministically over the contents.  Uses the same hash namespace as 'dir' formats already follow.


v0.1 -- bab8ac56e789b9d1833c0e2cf6a1712b84a8716c -- 27 Mar 2015
---------------------------------------------------------------

First demonstrable executors -- multiple implementations, with pluggable and swappable behavior from day one -- making it possible to cover the complete path of setup-snapshots => run-computation => take-new-snapshots.

- Feature: Prototype isolation using chroots.  Chroots provide filesystem isolation only, but are easy to use and require essentially zero setup -- works out of box.
- Feature: Prototype isolation using nsinit (part of the libcontainer project).  Some manual setup is required for nsinit to work.
- Internal: Improve code reuse for several common filesystem manipulations.


v0.0 -- cd19926ceaf7343cfe898c653cf6747ec02b070e -- 11 Mar 2015
---------------------------------------------------------------

Baby's First Determinism.  Laying down the groundwork for consistent data identity is the first step to repeatable computation and auditable, reproducible results.

- Feature: Formulas as a task and environment description system.
- Feature: Use regular directories as inputs, validating them using deterministic hashing -- specifying data this way makes it possible to guarantee identity and integrity, and do it without any concept of *where* the data is from or who hosts it.
- Internal: Tree traversal, metadata definitions, and bucket API for walking sections of a tree and accumulating hash info to produce deterministic IDs for a filesystem.
