package git

import (
	"encoding/hex"
	"net/url"

	"go.polydawn.net/repeatr/rio"
)

func mustBeFullHash(hash rio.CommitID) {
	if len(hash) != 40 {
		panic("gimme the whole thing")
	}
	if _, err := hex.DecodeString(string(hash)); err != nil {
		panic("git commit hashes are hex strings")
	}
}

/*
	Return a string that's safe to use as a dir name.

	Uses URL query escaping so it remains roughly readable.
	Does not attempt any URL normalization.
*/
func slugifyRemote(remoteURL string) string {
	return url.QueryEscape(remoteURL)
}
