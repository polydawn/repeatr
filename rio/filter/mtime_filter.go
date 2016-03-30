package filter

import (
	"time"

	"polydawn.net/repeatr/lib/fs"
)

var _ Filter = MtimeFilter{}

type MtimeFilter struct {
	Value time.Time
}

func (f MtimeFilter) Filter(attribs fs.Metadata) fs.Metadata {
	attribs.ModTime = f.Value
	return attribs
}
