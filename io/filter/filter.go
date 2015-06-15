/*
	Filters are used for normalizing/flattening attributes of filesystems,
	so that we can discard uninteresting pieces of information, or flatten
	them for the sake of consistency.

	One example of a useful filter is the "mtime" filter, which strips all
	the "modified time" properties from files -- since executing most
	applications results in a spray of unique and unrepeatable mtime properties,
	it's usually best to discard them.

	Typically, filters should be applied on output scans -- it makes sense
	to do data normalization *before* warehousing, both on general principle,
	and because it will make data deduplicate better.  However, filters
	may also be used on input/materialization; this may be useful in
	circumstances where the data committed to warehouse has various UIDs,
	but the local environment wants to get them all as the current UID.
*/
package filter

import (
	"polydawn.net/repeatr/lib/fs"
)

type Filter interface {
	Filter(fs.Metadata) fs.Metadata
}
