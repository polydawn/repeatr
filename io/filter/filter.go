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

/*
	Keeps a bunch of filters.

	Refers to some of them by name instead of just a slice of interfaces,
	because frequently transmats, the code that actually consumes filters,
	may have custom interactions with known types of filters for efficiency
	reasons (e.g. if uid is filtered, we might be able to just skip calling
	chown entirely, which means the whole system can run with lower privs).
*/
type FilterSet struct {
	Mtime MtimeFilter
	Uid   UidFilter
	Gid   GidFilter
}

// maybe have an applyAll method, and transmats that did Something Special just nil the relevant ones and then call that?

func (fs FilterSet) Put(filt Filter) FilterSet {
	switch f2 := filt.(type) {
	case MtimeFilter:
		fs.Mtime = f2
	case UidFilter:
		fs.Uid = f2
	case GidFilter:
		fs.Gid = f2
	}
	return fs
}
