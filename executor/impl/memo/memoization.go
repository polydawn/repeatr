package memo

import (
	"os"

	. "github.com/polydawn/go-errcat"
	"github.com/polydawn/refmt"
	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/rio/fs"
)

/*
	Attempt to load a memoized runRecord.

	If it doesn't exist (or memoDir is zero entirely), returns nil nil.
*/
func loadMemo(setupHash api.SetupHash, memoDir fs.AbsolutePath) (rr *api.RunRecord, err error) {
	// Try to open file.
	f, err := os.Open(memoPath(setupHash, memoDir).String())
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
	rr = &api.RunRecord{}
	if err := refmt.NewUnmarshallerAtlased(json.DecodeOptions{}, f, api.RepeatrAtlas).Unmarshal(rr); err != nil {
		return nil, Errorf(repeatr.ErrLocalCacheProblem, "error parsing memo for setupHash %q: %s", setupHash, err)
	}
	return rr, nil
}

func saveMemo(setupHash api.SetupHash, memoDir fs.AbsolutePath, rr *api.RunRecord) error {
	// Open file.
	f, err := os.OpenFile(memoPath(setupHash, memoDir).String(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return Errorf(repeatr.ErrLocalCacheProblem, "could not save memo: %s", err)
	}
	defer f.Close()
	// Write.
	if err := refmt.NewMarshallerAtlased(json.EncodeOptions{}, f, api.RepeatrAtlas).Marshal(rr); err != nil {
		return Errorf(repeatr.ErrLocalCacheProblem, "could not save memo: %s", err)
	}
	return nil
}

func memoPath(setupHash api.SetupHash, memoDir fs.AbsolutePath) fs.AbsolutePath {
	// REVIEW should probably use the threesplits too I guess?  unclear how far this needs to scale.
	return memoDir.Join(fs.MustRelPath(string(setupHash)))
}
