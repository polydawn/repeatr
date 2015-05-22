/*
	Filters are used for normalizing/flattening attributes of filesystems,
	so that we can discard uninteresting pieces of information, or flatten
	them for the sake of consistency.

	One example of a useful filter is the "mtime" filter, which strips all
	the "modified time" properties from files -- since executing most
	applications results in a spray of unique and unrepeatable mtime properties,
	it's usually best to discard them.
*/
package filter

import (
	"polydawn.net/repeatr/lib/fs"
)

type Filter interface {
	Filter(fs.Metadata) fs.Metadata
}
