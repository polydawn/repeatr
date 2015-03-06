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
	// is the next one still a child?
	next := *i.that + 1
	if next >= len(i.lines) {
		return nil
	}
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
