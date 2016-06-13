package rio

import (
	"fmt"

	"github.com/spacemonkeygo/errors"
)

/*
	Groups all errors emitted by the Repeatable Input/Output systems.

	Roughly, errors are categorized by what part of the system hit the
	problem (and thus who needs to be involved in fixing settings, etc).

		- Error
		  - ConfigError (the user's fault)
		  - TransmatError (something went wrong internally; maybe the repeatr daemon hit rough permissions, etc)
		  - WarehouseError
		    - WarehouseConnectionError (warehouses are often remote; network errors deserve their own heading)
		    - DataDNE (not necessarily a panic-worthy offense)
		    - HashMismatchError (this is somewhere between extremely rudely failing component and outright malice)
		  - MoorError (from either materialize or scan: ran out of space, didn't have perms, something)

	All of these are errors that can be raised from using transmats.
	(They're not all grouped under TransmatError because they're not all
	the transmat's fault, per se; fixing them requires looking afield.)

	Additionally, these other systems may fail, but are rare, internal, and serious:
		- Error
		  - PlacerError
		  // actually that's it... assemblers, to date, don't represent enough
		  //  code to have their own interesting failure modes.

*/
var Error *errors.ErrorClass = errors.NewClass("IntegrityError") // grouping, do not instantiate // n.b. the ambiguity and alarmingness of this error name is the clearest example of why this package needs rethinking on the name.

/*
	Raised to indicate that some configuration is missing or malformed.
*/
var ConfigError *errors.ErrorClass = Error.NewClass("ConfigError")

/*
	Raised to indicate a serious internal error with a transmat's functioning.

	Typically these indicate a need for the whole program to stop; examples are
	the repeatr daemon hitting permissions problems in the main work area,
	or running out of disk space, or something equally severe.

	Errors relating to the either the warehouse, the data integrity checks,
	or the operational theater on the local filesystem are all different
	categories of error.
*/
var TransmatError *errors.ErrorClass = Error.NewClass("TransmatError")

/*
	Raised to indicate problems getting data from a storage warehouse.

	The error may or may not be temporary, depending on the subtype.
*/
var WarehouseError *errors.ErrorClass = Error.NewClass("WarehouseError")

/*
	Raised when a warehouse is not available.

	The warehouse may not exist, or may be offline, or there may be a network
	interruption, etc.
*/
var WarehouseUnavailableError *errors.ErrorClass = WarehouseError.NewClass("WarehouseUnavailableError")

/*
	Raised when there are unexpected connectivity issues with a storage warehouse.
*/
var WarehouseIOError *errors.ErrorClass = WarehouseError.NewClass("WarehouseIOError")

/*
	Raised when there are clearly corrupt contents read from a warehouse.

	This is distinct from a HashMismatchError in that it represents some
	form of failure to parse data before we have even reached the stage
	where the content's full semantic hash is computable (for example,
	with a tar transmat, if the tar header is completely nonsense, we
	just have to give up).
*/
var WarehouseCorruptionError *errors.ErrorClass = WarehouseError.NewClass("WarehouseCorruptionError")

/*
	Raised when requested data is not available from a storage warehouse.

	This is not necessarily a panic-worthy offense, but may be raised as a panic
	anyway by e.g. `Materialize` methods, since they're expressing an expectation
	that we're *going* to get that data.
*/
var DataDNE *errors.ErrorClass = WarehouseError.NewClass("DataDoesNotExistError")

/*
	Raised when data fails to pass integrity checks.

	This means there have been data integrity issues in the storage or
	transport systems involved -- either the storage warehouse has
	experienced corruption, or the transport is having reliability
	issues, or, this may be an active attack (i.e. MITM).
*/
var HashMismatchError *errors.ErrorClass = WarehouseError.NewClass("HashMismatchError")

func NewHashMismatchError(expectedHash, actualHash string) *errors.Error {
	return HashMismatchError.NewWith(
		fmt.Sprintf("expected hash %q, got %q", expectedHash, actualHash),
		errors.SetData(HashExpectedKey, expectedHash),
		errors.SetData(HashActualKey, actualHash),
	).(*errors.Error)
}

// Found on `InputHashMismatchError`
var HashExpectedKey errors.DataKey = errors.GenSym()

// Found on `InputHashMismatchError`
var HashActualKey errors.DataKey = errors.GenSym()

/*
	Raised to indicate problems working on the operational theater on
	the local filesystem (e.g. permission denied to read during a `Scan`
	or permission denied or out-of-space during a write during `Materialize`).
*/
var AssemblyError *errors.ErrorClass = Error.NewClass("AssemblyError")

/*
	Wraps any other unknown errors just to emphasize the system that raised them;
	any well known errors should use a different type.

	If an error of this type is exposed to the user, it should be
	considered a bug, and specific error detection added to the site.
*/
var UnknownError *errors.ErrorClass = Error.NewClass("IntegrityUnknownError")