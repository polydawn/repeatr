package filter

import (
	"polydawn.net/repeatr/lib/fs"
)

var _ Filter = UidFilter{}

type UidFilter struct {
	Value int
}

func (f UidFilter) Filter(attribs fs.Metadata) fs.Metadata {
	attribs.Uid = f.Value
	return attribs
}
