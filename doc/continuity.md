Commissions, Catalogs, and Continuity ohmy!
===========================================

Managing data comes in three layers: « *what* » it is, « *where* » it is, and « *why* » it is.

- The **what** is covered by the hash, in a content-addressible system.
  [Formulas](./formulas.md) are already totally on top of this part.
- The **where** is the URL to fetch things from.
  Again, totally covered already by [Formula](./formulas.md) input/output config.
- The **why**...

Well, the **why** is where things get fun :)
In Repeatr, this is where `Catalog`s and `Commission`s come in.

Catalogs connect human-meaningful names to content-addressible hashes.
Commissions are structurally almost identical for Formulas, but refer to the names defined by catalogs instead of directly pinning ware hashes,
and thus can (well) *commission* an entire sequence of Formulas as the Catalogs publish new content versions under existing names.
