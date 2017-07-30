/*
	Interfaces of rio commands.

	The heuristic for the rio callable library API is that essentially
	all information must be racked up in the call already: the assumption
	is that the side doing the real work might not share a filesystem with
	you (well, in rio's case, it probably does!  but it might be a subset,
	translated through chroots and bind mounts), doesn't share env vars, etc.
	So, general rule of thumb: the caller is going to have already handled
	all config loading and parsing, and those objects are params in this funcs.
*/
package rio

import (
	"context"

	"go.polydawn.net/repeatr/api"
)

type UnpackFunc func(
	ctx context.Context, // Long-running call.  Cancellable.
	wareID api.WareID, // What wareID to fetch for unpacking.
	path string, // Where to unpack the fileset (absolute path).
	filters api.FilesetFilters, // Optionally: filters we should apply while unpacking.
	warehouses []api.WarehouseAddr, // Warehouses we can try to fetch from.
	monitor Monitor, // Optionally: callbacks for progress monitoring.
) (api.WareID, error)

type PackFunc func(
	ctx context.Context, // Long-running call.  Cancellable.
	path string, // The fileset to scan and pack (absolute path).
	filters api.FilesetFilters, // Optionally: filters we should apply while unpacking.
	warehouse api.WarehouseAddr, // Warehouse to save into (or blank to just scan).
	monitor Monitor, // Optionally: callbacks for progress monitoring.
) (api.WareID, error)

type MirrorFunc func(
	ctx context.Context, // Long-running call.  Cancellable.
	wareID api.WareID, // What wareID to mirror.
	target api.WarehouseAddr, // Warehouse to ensure the ware is mirrored into.
	sources []api.WarehouseAddr, // Warehouses we can try to fetch from.
	monitor Monitor, // Optionally: callbacks for progress monitoring.
) (api.WareID, error)

type Monitor struct {
	/*
		Callback for notifications about progress updates.

		Imagine it being used to draw the following:

			Frobnozing (145/290kb): [=====>    ]  50%

		The 'totalProg' and 'totalWork' ints are expected to be a percentage;
		when they equal, a "done" state should be up next.
		A value of 'totalProg' greater than 'totalWork' is nonsensical.

		The 'phase' and 'desc' args are freetext;
		Typically, 'phase' will remain the same for many calls in a row, while
		'desc' is used to communicate a more specific contextual info
		than the 'total*' ints and like the ints may likely change on each call.
	*/
	NotifyFn func(phase, desc string, totalProg, totalWork int)
}
