package jank

import (
	"os"
)

// TODO replace this with a more granular set of options
// TODO and have it pass down explicitly, all the way, from the CLI just like other mature things do

/*
	Return the home-base path prefix that this process will cram ALL state under.

	Consumers of this space include:

		- all caching for the `rio` system
		- assembler working space (e.g. for COW filesystems)
		- executor working space (for the rootfs to be materialized, and any config-passing files)
		- the "assets" cache

	Usually it's `"/var/lib/repeatr"`, but it can be overriden by the `REPEATR_BASE`
	environment variable.  (The test system uses this to get pick a single prefix
	to invoke a group of package tests to run together on the same state,
	while making certain nothing survives to interfere between runs.)
*/
func Base() string {
	base := os.Getenv("REPEATR_BASE")
	if base == "" {
		base = "/var/lib/repeatr"
	}
	os.MkdirAll(base, 0755)
	return base
}
