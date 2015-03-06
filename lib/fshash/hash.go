package fshash

import (
	"hash"

	"polydawn.net/repeatr/lib/treewalk"
)

func Hash(bucket Bucket, hasher hash.Hash) ([]byte, error) {
	preVisit := func(node treewalk.Node) error {
		record := node.(RecordIterator).Record()
		metabin, err := record.Metadata.MarshalBinary()
		if err != nil {
			return err
		}
		// TODO : this also needs some higher level alignment/length stuff
		_, err = hasher.Write(metabin)
		if err != nil {
			return err
		}
		_, err = hasher.Write(record.ContentHash)
		return err
	}
	postVisit := func(node treewalk.Node) error { return nil }
	if err := treewalk.Walk(bucket.Iterator(), preVisit, postVisit); err != nil {
		// TODO: these actually all seem like really severe panic-worthy errors
		return nil, err
	}
	return hasher.Sum(nil), nil
}
