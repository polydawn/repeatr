/*
	NOTE YE WELL: this is a placeholder package,
	wherein we're mirroring many types declared in repeatr.

	We're evolving them freely and independently for the moment, but
	the time will come when we have to make both projects line up again!
*/
package rdef

import (
	"fmt"
	"strings"

	"github.com/polydawn/refmt/obj/atlas"
)

/*
	Ware IDs are content-addressable, cryptographic hashes which uniquely identify
	a "ware" -- a packed filesystem snapshot.

	Ware IDs are serialized as a string in two parts, separated by a colon --
	for example like "git:f23ae1829" or "tar:WJL8or32vD".
	The first part communicates which kind of packing system computed the hash,
	and the second part is the hash itself.
*/
type WareID struct {
	Type string
	Hash string
}

func ParseWareID(x string) (WareID, error) {
	ss := strings.SplitN(x, ":", 2)
	if len(ss) < 2 {
		return WareID{}, fmt.Errorf("wareIDs must have contain a colon character (they are of form \"<type>:<hash>\")")
	}
	return WareID{ss[0], ss[1]}, nil
}

func (x WareID) String() string {
	return x.Type + ":" + x.Hash
}

var WareID_AtlasEntry = atlas.BuildEntry(WareID{}).Transform().
	TransformMarshal(atlas.MakeMarshalTransformFunc(
		func(x WareID) (string, error) {
			return x.String(), nil
		})).
	TransformUnmarshal(atlas.MakeUnmarshalTransformFunc(
		func(x string) (WareID, error) {
			return ParseWareID(x)
		})).
	Complete()

type AbsPath string // Identifier for output slots.  Coincidentally, a path.

type (
	Formula struct {
		Inputs  FormulaInputs
		Action  FormulaAction
		Outputs FormulaOutputs
	}

	FormulaInputs map[AbsPath]WareID

	FormulaOutputs map[AbsPath]string // TODO probably need more there than the ware type name ... although we could put normalizers in the "action" section

	/*
		Defines the action to perform to evaluate the formula -- some commands
		or filesystem operations which will be run after the inputs have been
		assembled; the action is done, the outputs will be saved.
	*/
	FormulaAction struct {
		// An array of strings to hand as args to exec -- creates a single process.
		//
		// TODO we want to add a polymorphic option here, e.g.
		// one of 'Exec', 'Script', or 'Reshuffle' may be set.
		Exec []string
	}

	SetupHash string // HID of formula
)

var (
	Formula_AtlasEntry       = atlas.BuildEntry(Formula{}).StructMap().Autogenerate().Complete()
	FormulaAction_AtlasEntry = atlas.BuildEntry(FormulaAction{}).StructMap().Autogenerate().Complete()
)

type RunRecord struct {
	UID       string             // random number, presumed globally unique.
	Time      int64              // time at start of build.
	FormulaID SetupHash          // HID of formula ran.
	Results   map[AbsPath]WareID // wares produced by the run!

	// --- below: addntl optional metadata ---

	Hostname string            // hostname.  not a trusted field, but useful for debugging.
	Metadata map[string]string // escape valve.  you can attach freetext here.
}

var RunRecord_AtlasEntry = atlas.BuildEntry(RunRecord{}).StructMap().Autogenerate().Complete()

type RunRecordHash string // HID of RunRecord.  Includes UID, etc, so quite unique.  Prefer this to UID for primary key in storage (it's collision resistant).
