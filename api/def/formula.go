package def

import (
	"crypto/sha512"

	"github.com/ugorji/go/codec"
)

/*
	Formula describes `action(inputs) -> (outputs)`.
*/
type Formula struct {
	Inputs  InputGroup  `json:"inputs"`
	Action  Action      `json:"action"`
	Outputs OutputGroup `json:"outputs"`
}

/*
	Hash the formula -- including the inputs, actions, and output slot specs;
	excluding any actual output ware hashes, and excluding any non-conjecture-worthy
	bits like warehouse coordinates from both the input and output sides.

	Caveat Emptor: this definition is should be treated as a proposal, not blessed.
	Future versions may change the exact serialization used, and thus may not
	map into the same strings as previous versions.

	The returned string is the base58 encoding of a SHA-384 hash, though
	there is no reason you should treat it as anything but opaque.
	The returned string may be relied upon to be all alphanumeric characters.
	FIXME actually use said encoding.
*/
func (f Formula) Hash() string {
	// Copy and zero other things that we don't want to include in canonical IDs.
	// This is working around lack of useful ways to pass encoding style hints down
	//  with our current libraries.
	f2 := f.Clone()
	for _, spec := range f2.Inputs {
		spec.Warehouses = nil
	}
	for _, spec := range f2.Outputs {
		spec.Hash = ""
		spec.Warehouses = nil
	}
	// Hash the rest, and thar we be.
	hasher := sha512.New384()
	codec.NewEncoder(hasher, &codec.CborHandle{}).MustEncode(f2)
	return b58encode(hasher.Sum(nil))
}
