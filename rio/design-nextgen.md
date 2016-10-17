rio & transmat design: next gen
===============================

Todos for the next big refactor wave, whenever that comes.

re-split
--------

Some original abstraction choices could use review entirely, and naming around
whether an object/interface is for local management or remote tracking could be clearer.

- remote warehouses: give them their own full interface.
  - this has arisen consistently in every transmat to date; enshrine it.
- local shelving: give them their own interface and names.
  - current generation of code makes it too easy to confuse local/remote and makes error handling require too many neurons as a result.

Other generalizations worth following:

- Always retain the user-specified warehouseCoords.  Pass them down the whole way, even if not technically necessary.
  - e.g. don't just keep the internally parsed coord info; echoing the user is better for consistency in logging and error messages.

transmats
---------

- lots of the protocols underneath are K/V in essense
  - this should be extracted out to mixin'able behaviors, would save a lot of code
  - 'http://', 'file://', even 's3://' and 'gcs://'... we usually treat them as K/V.
    - which is not to say we *have* to.  transmat impls would still choose what mixins they're using how.
	  - for example the 'dir' transmat uses 'file://' very differently than the tar transmat does, and that's OK.
  - might even be a mode where we treat ipfs as K/V with the same hash schemas
    - this weirds me out a lil, but otoh works really well to abstract us from their dedup chunking choices
  - the 'writeController' stuff started in the dir transmat is approximately The Right Thing... 
    - now it just needs to be extracted and made into a real interface with multiple implementations.

- most of the arena impls are highly duplicated
  - same thing, should just make them mixins
  - there's probably just a file one and a dir one at this point

- we might come out looking clearer if the local unpack stuff was a parameter to transportation just like remote warehouse coords are
  - right now it's implied that the transmat objects are owners of that

- should we promote the "caching" behavior to a more serious name?
  - it's a little amusing that the local CAS behavior is only applied by the cacher.  reasonable.  but amusing.
  - what if we want a CAS asset store, unpacked, locally?  e.g. exactly like repeatr assets does.
    - we should have a one-word way to do that.
	- this is distinct from transmat mirroring because it's *unpacked* (like the cache is).

filters
-------

- lack a clear distinction between serializable config and live API.  make that distinct.
  - remove the cheneyian options patterns.  they're great for purely programmatic apis; terrible for ones that are supposed to serialize.
