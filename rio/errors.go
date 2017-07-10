package rio

import (
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
)

/*
	Raised to indicate a serious internal error with a transmat's functioning.

	Typically these indicate a need for the whole program to stop; examples are
	the repeatr daemon hitting permissions problems in the main work area,
	or running out of disk space, or something equally severe.

	Errors relating to communicating with a warehouse, data integrity checks,
	absense of data, etc, are all recoverable errors, and are expressed with
	error types exported in the `api/def` package.
*/
type ErrInternal struct {
	Msg string
	meep.TraitAutodescribing
	meep.TraitCausable
	meep.TraitTraceable
}

/*
	Raised to indicat a serious error while assembling filesystems together.

	Typically these indicate a need for the whole program to stop; examples are
	the repeatr daemon hitting permissions problems in the main work area,
	or running out of disk space, or something equally severe.

	(This is not significantly different than `ErrInternal`,
	but indicates which phase of work the error came from.
	Placer and assembler functions will raise this error; transmats won't.)
*/
type ErrAssembly struct {
	meep.TraitAutodescribing
	meep.TraitCausable
	meep.TraitTraceable
	System string // e.g. "copyingplacer" or etc.
	Path   string // often just "srcPath" or "destPath"
	// "op" is usually covered in the io error string, if that's a cause
}

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.

	If an error of this type is exposed to the user, it should be
	considered a bug, and specific error detection added to the site.
*/
type ErrUnknown struct {
	meep.TraitAutodescribing
	meep.TraitCausable
	meep.TraitTraceable
}

/*
	A standard TryPlan snippet for passing up any well-known error types
	which are reasonable for a transmat to raise during operation,
	and swaddling anything else in an ErrUnknown, so it gets a stack from here.
*/
var TryPlanWhitelist = meep.TryPlan{
	{ByType: &ErrInternal{}, Handler: func(e error) { panic(e) }},
	{ByType: &ErrAssembly{}, Handler: func(e error) { panic(e) }},
	{ByType: &ErrUnknown{}, Handler: func(e error) { panic(e) }},
	{ByType: &def.ErrWarehouseUnavailable{}, Handler: func(e error) { panic(e) }},
	{ByType: &def.ErrWarehouseProblem{}, Handler: func(e error) { panic(e) }},
	{ByType: &def.ErrWareDNE{}, Handler: func(e error) { panic(e) }},
	{ByType: &def.ErrHashMismatch{}, Handler: func(e error) { panic(e) }},
	{ByType: &def.ErrWareCorrupt{}, Handler: func(e error) { panic(e) }},
	{CatchAny: true, Handler: meep.TryHandlerMapto(&ErrUnknown{})},
}
