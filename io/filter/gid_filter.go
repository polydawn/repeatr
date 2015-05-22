package filter

import (
	"polydawn.net/repeatr/lib/fs"
)

var _ Filter = GidFilter{}

type GidFilter struct {
	Value int
}

func (f GidFilter) Filter(attribs fs.Metadata) fs.Metadata {
	attribs.Gid = f.Value
	return attribs
}
