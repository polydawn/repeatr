package formula

import (
	"crypto/sha512"
	"encoding/base64"

	"github.com/ugorji/go/codec"

	"go.polydawn.net/repeatr/api/def"
)

// These types are all aliases for the same thing:
// we keep the types attached as a hint of how far along they are.
// (Even though they're structurally the same, their semantics change.)

type Commission struct {
	ID CommissionID
	def.Formula
}

type Stage2 def.Formula

type Stage3 def.Formula

type CommissionID string

type Stage2ID string

type Stage3ID string

func (f Stage2) ID() string {
	// San check empty values.  programmer error if set.
	for _, spec := range f.Outputs {
		if spec.Hash != "" {
			panic("stage2 formula with output hash set")
		}
	}
	// Copy and zero other things that we don't want to include in canonical IDs.
	// This is working around lack of useful ways to pass encoding style hints down
	//  with out current libraries.
	f2 := def.Formula(f).Clone()
	for _, spec := range f2.Inputs {
		spec.Warehouses = nil
	}
	for name, spec := range f2.Outputs {
		if spec.Conjecture == false {
			delete(f2.Outputs, name)
		}
		spec.Warehouses = nil
	}
	// hash the rest, and thar we be
	hasher := sha512.New384()
	codec.NewEncoder(hasher, &codec.CborHandle{}).MustEncode(f2)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}
