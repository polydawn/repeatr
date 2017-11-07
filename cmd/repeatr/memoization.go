package main

import (
	"os"
	"path/filepath"

	. "github.com/polydawn/go-errcat"
	"github.com/polydawn/refmt"
	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
)

/*
	Attempt to load a memoized runRecord.

	If it doesn't exist (or memoDir is zero entirely), returns nil nil.
*/
func loadMemo(setupHash api.SetupHash, memoDir string) (rr *api.RunRecord, err error) {
	// Nil result if no memodir.  That's fine.
	if memoDir == "" {
		return nil, nil
	}

	// Try to open file.
	// REVIEW should probably use the threesplits too I guess?  unclear how far this needs to scale.
	// REVIEW should we make sure this can jive with hitch runrecord layouts?  (which are not one-to-many with the setupHash?)
	f, err := os.Open(filepath.Join(memoDir, string(setupHash)))
	if err != nil {
		// If not exists, no memo.  Fine.
		if os.IsNotExist(err) {
			return nil, nil
		}
		// Any other error is worth warning the caller about.
		return nil, Errorf(repeatr.ErrLocalCacheProblem, "error reading memodir: %s", err)
	}
	defer f.Close()

	// Read and return the memoized runrecord.
	if err := refmt.NewUnmarshallerAtlased(json.DecodeOptions{}, f, api.RepeatrAtlas).Unmarshal(rr); err != nil {
		return nil, Errorf(repeatr.ErrLocalCacheProblem, "error parsing memo for setupHash %q: %s", setupHash, err)
	}
	return rr, nil
}
