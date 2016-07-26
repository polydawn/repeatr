package filter

import (
	"go.polydawn.net/repeatr/lib/fs"
)

// TODO While the user is allowed to do whatever they want, somewhere up near
//  the config layer we should probably detect if any inputs use uid/gid
//  filtering to avoid chowns, and if that formula doesn't also spec uid/gid
//  filtering to normalize the outputs, we should emit a helpful warning message.

var _ Filter = UidFilter{}

type UidFilter struct {
	Value int
}

func (f UidFilter) Filter(attribs fs.Metadata) fs.Metadata {
	attribs.Uid = f.Value
	return attribs
}
