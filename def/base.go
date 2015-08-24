package def

import (
	"os"
)

/*
	Return the home-base path prefix that this process will cram ALL state under.

	Usually it's `"/tmp/repeatr"`, but it can be overriden by the `REPEATR_BASE`
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
