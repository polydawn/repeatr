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


Catalogs
--------

Catalogs contain a bunch of names for Wares.
Catalogs are composed of a one or more "tracks",
and each track is an ordered list of "releases", going from newest to oldest.

Tracks can be used to name a series of releases for the purpose of smooth
updating; e.g. the "4.x" series vs the "5.x" series would be two different
tracks.  Tracks bundled within the same Catalog share an author (and
signing authority).

Here's a sketch of an example catalog structure.
Note that it's perfectly normal for different tracks to refer to the same ware.

```
Catalog{
    "stable": Track{
        Releases[
            {ware:"a3vcj3"},
            {ware:"b56jql"},
        ]
    }
    "4.x": Track{
        Releases[
            {ware:"b56jql"},
        ]
    }
}
```

Commissions
-----------

Commissions have the same basic structural outline as [Formulas](./formulas.md) --
inputs, actions, and outputs:

```
inputs:
    "/":
        catalog: "staid-ubuntu:14.04"
    "/app/whizbang/":
        catalog: "whizbangery:stable"
action:
    command: ["/app/whizbang/bin/buildr"]
outputs:
    "/task/output":
        catalog: "phase-2:unstable"
```

A Commission can be resolved against a library of Catalogs to produce a Formula.
When the Commission refers to a catalog, or a specific track inside the catalog,
the most recent releases in a track will automatically selected.

Since Catalogs retain their own historical versions, this process is reversible:
given a Commission, a Formula, and the library of Catalogs
(even if the catalogs are now more up-to-date than when the Formula was originally resolved),
we can check if the Formula is a valid resolution of the Commission.
