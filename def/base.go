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
		base = "/tmp/repeatr" // change to var/lib or something when we feel more serious
	}
	os.MkdirAll(base, 01755)
	return base
}