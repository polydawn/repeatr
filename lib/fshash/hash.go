package fshash

import (
	"hash"

	"github.com/ugorji/go/codec"
	"polydawn.net/repeatr/lib/treewalk"
)

func Hash(bucket Bucket, hasher hash.Hash) ([]byte, error) {
	enc := codec.NewEncoder(hasher, new(codec.CborHandle))
	hasher.Write([]byte{codec.CborStreamArray})

	preVisit := func(node treewalk.Node) error {
		record := node.(RecordIterator).Record()
		record.Metadata.Marshal(hasher)
		// TODO : this also needs some higher level alignment/length stuff
		_ = enc
		_, err := hasher.Write(record.ContentHash)
		return err
	}
	postVisit := func(node treewalk.Node) error { return nil }
	if err := treewalk.Walk(bucket.Iterator(), preVisit, postVisit); err != nil {
		// TODO: these actually all seem like really severe panic-worthy errors
		return nil, err
	}
	hasher.Write([]byte{0xff}) // should be `codec.CborStreamBreak` but upstream has an export bug :/
	return hasher.Sum(nil), nil
}
