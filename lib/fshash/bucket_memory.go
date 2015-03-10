package fshash

import (
	"sort"
	"strings"

	"polydawn.net/repeatr/lib/treewalk"
)

var _ Bucket = &MemoryBucket{}

type MemoryBucket struct {
	// my kingdom for a red-black tree or other sane sorted map implementation
	lines []Record
}

func (b *MemoryBucket) Record(metadata Metadata, contentHash []byte) {
	b.lines = append(b.lines, Record{metadata, contentHash})
}

/*
	Get a `treewalk.Node` that starts at the root of the bucket.
	The walk will be in deterministic, sorted order (and thus is appropriate
	for hashing).

	This is only safe for non-concurrent use and depth-first traversal.
	If the data structure is changed, or (sub)iterators used out of order,
	behavior is undefined.
*/
func (b *MemoryBucket) Iterator() RecordIterator {
	sort.Sort(linesByFilepath(b.lines))
	// TODO: check for rootedness
	var that int
	return &memoryBucketIterator{b.lines, 0, &that}
}

type memoryBucketIterator struct {
	lines []Record
	this  int  // pretending a linear structure is a tree is weird.
	that  *int // this is the last child walked.
}

func (i *memoryBucketIterator) NextChild() treewalk.Node {
	// Since we sorted before starting iteration, all child nodes are contiguous and follow their parent.
	// Each treewalk node keeps its own record's index (implicitly, this is forming a stack),
	// and they all share the same value for last index walked, so when a child has been fully iterated over,
	// the next call on the parent will start looking right after all the child's children.
	next := *i.that + 1
	if next >= len(i.lines) {
		return nil
	}
	// is the next one still a child?
	if strings.HasPrefix(i.lines[next].Metadata.Name, i.lines[i.this].Metadata.Name+"/") {
		*i.that = next
		// TODO: check for missing trees
		// TODO: check for repeated names
		return &memoryBucketIterator{i.lines, *i.that, i.that}
	}
	return nil
}

func (i memoryBucketIterator) Record() Record {
	return i.lines[i.this]
}
