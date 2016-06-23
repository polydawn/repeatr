Glossary
========

**ware**: A piece of data.  Typically a filesystem tree.  (A `.tar` file can handled as a ware.)  Wares are stored content-addressably.

**content-addressable**: The practice of giving data a name based on its own content.  Typically implemented by using a cryptographic hash over the content.  Content-addressable systems are immutable.

**formula**: A document describing a series of `ware`s, how to arrange them in a filesystem, some action to perform on them, and what parts of the filesystem to save as resultant `ware`s.  `Ware`s in a formula are referred to by their content-addressable hash, meaning formulas in turn are an immutable description of how to set up and run something.  Repeatr evaluates formulas.

**catalog**: A named record pointing to one or more `ware`s.  Catalogs associate the name to the `ware`'s hash, and are usually cryptographically signed.  Catalogs are a mutable structure, but also  continue to carry references to previously-referenced `ware`s even when updated.

**commission**: A document naming a series of `catalog`s, how to arrange their referenced `ware`s in a filesystem, some action to perform on them, and which `catalog`s should be updated to refer to new `ware`s saved from parts of the resultant filesystem.  In other words -- like formulas, but connected to catalogs instead of directly to wares.
