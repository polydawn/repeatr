package def

import (
	"time"
)

/*
	Filters are transformations that can be applied to data, either to
	normalize it for storage or to apply attributes to it before feeding
	the data into an action's inputs.

	The following filters are available:

		- uid   -- the posix user ownership number
		- gid   -- the posix group ownership number
		- mtime -- the posix file modification timestamp

	'uid', 'gid', and 'mtime' are all filtered by default on formula outputs --
	most use cases do not need these attributes, and they are a source of nondeterminism.
	If you want to keep them, you may specify	`uid keep`, `gid keep`, `mtime keep`,
	etc; if you want the filters to flatten to different values than the defaults,
	you may specify `uid 12000`, etc.
	(Note that the default mtime filter flattens the time to Jan 1, 2010 --
	*not* epoch.  Some contemporary software has been known to regard zero/epoch
	timestamps as errors or empty values, so we've choosen a different value in
	the interest of practicality.)

	Filters on inputs will be applied after the data is fetched and before your
	job starts.
	Filters on outputs will be applied after your job process exits, but before
	the output hash is computed and the data committed to any warehouses for storage.

	Note that these filters are built-ins (and there are no extensions possible).
	If you need more complex data transformations, incorporate it into your job
	itself!  These filters are built-in because they cover the most common sources
	of nondeterminism, and because they are efficient to implement as special
	cases in the IO engines (and in some cases, e.g. ownership filters, are also
	necessary for security properties an dusing repeatr IO with minimal host
	system priviledges).
*/
type Filters struct {
	UidMode   FilterMode
	Uid       int
	GidMode   FilterMode
	Gid       int
	MtimeMode FilterMode
	Mtime     time.Time
}

type FilterMode int

const (
	FilterUninitialized FilterMode = iota
	FilterUse
	FilterKeep
	FilterHost
)

var (
	FilterDefaultUid   = 1000
	FilterDefaultGid   = 1000
	FilterDefaultMtime = time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)
)
