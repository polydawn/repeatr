package def

import (
	"fmt"
)

/*
	Error raised when parsing configuration or formulas
	(generally, something user-input).

	Basic parsing errors like "got number, expected string" are included in this,
	but not more advanced semantics (e.g. "overlapping mounts" is a semantic
	error, not a parsing one, so isn't raised as this type).
*/
type ErrConfigParsing struct {
	Key         string
	Msg         string
	MustBe      string
	WasActually string
}

func (e ErrConfigParsing) Error() string {
	return e.Msg
}

func newConfigValTypeError(key, mustBe, wasActually string) error {
	msg := fmt.Sprintf("config key %q must be a %s; was %s", key, mustBe, wasActually)
	return ErrConfigParsing{
		Key:         key,
		Msg:         msg,
		MustBe:      mustBe,
		WasActually: wasActually,
	}
}

/*
	Error raised upon failure to validate semantics of a config object
	or formula or etc.
*/
type ErrConfigValidation struct {
	Key string
	Msg string
}

func (e ErrConfigValidation) Error() string {
	return e.Msg
}

/*
	Raised when a warehouse is not available.

	The warehouse may not exist, or may be offline, or there may be a network
	interruption, etc.
*/
type ErrWarehouseUnavailable struct {
	Msg    string         `json:"msg,omitempty"`  // may indicate things like "*all* offline" or "you didn't configure one, silly"
	From   WarehouseCoord `json:"from,omitempty"` // may be blank if it's the bulk check
	During string         `json:"during"`         // 'fetch' or 'save'
}

func (e ErrWarehouseUnavailable) Error() string {
	from := ""
	if e.From != "" {
		from = fmt.Sprintf(" from %s", e.From)
	}
	return fmt.Sprintf("Warehouse Unavailable for %s: %s%s", e.During, e.Msg, from)
}

/*
	Raised when there are problems communicating with a warehouse.

	IO issues after connection handshake, corruption of contents (e.g. malformed
	headers in the content (hash mismatch is its own thing)).
	Local warehouses (e.g. "file://...") may also raise these in the event
	of issues like out-of-disk when saving, etc.

	We'd probably use some nested `error` freeform instead of `Msg`, but since
	this type is one of the exported API errors, it must be unmarshallable without
	requiring further unbounded polymorphism shenanigans,
	so we keep it simple: string.

	REVIEW: undecided if this should be exported as an API error at all.
*/
type ErrWarehouseProblem struct {
	Msg    string         `json:"msg"`
	During string         `json:"during"` // 'fetch' or 'save'
	Ware   Ware           `json:"ware,omitEmpty"`
	From   WarehouseCoord `json:"from"`
}

func (e ErrWarehouseProblem) Error() string {
	regarding := ""
	if e.Ware.Hash != "" {
		regarding = fmt.Sprintf(" of %q", e.Ware.Hash)
	}
	return fmt.Sprintf("Warehouse errored during %s%s: %s, from %s", e.During, regarding, e.Msg, e.From)
}

/*
	Raised when requested data is not available from a storage warehouse.

	This is not necessarily a panic-worthy offense, but may be raised as a panic
	anyway by e.g. `Materialize` methods, since they're expressing an expectation
	that we're *going* to get that data.
*/
type ErrWareDNE struct {
	Ware Ware           `json:"ware"`
	From WarehouseCoord `json:"from,omitempty"`
}

func (e ErrWareDNE) Error() string {
	from := ""
	if e.From != "" {
		from = " from " + string(e.From)
	}
	return fmt.Sprintf("Ware DNE: hash %q not found%s", e.Ware.Hash, from)
}

/*
	Raised when data fails to pass integrity checks.

	This means there have been data integrity issues in the storage or
	transport systems involved -- either the storage warehouse has
	experienced corruption, or the transport is having reliability
	issues, or, this may be an active attack (i.e. MITM).
*/
type ErrHashMismatch struct {
	Expected Ware           `json:"expected"`
	Actual   Ware           `json:"actual"`
	From     WarehouseCoord `json:"from"`
}

func (e ErrHashMismatch) Error() string {
	return fmt.Sprintf("Hash Mismatch: expected %q, got %q from %s", e.Expected.Hash, e.Actual.Hash, e.From)
}

/*
	Raised when encountering clearly corrupt contents read from a warehouse.

	This is distinct from `ErrHashMismatch` in that it represents some
	form of failure to parse data before we have even reached the stage
	where the content's full semantic hash is computable (for example,
	with a tar transmat, if the tar header is completely nonsense, we
	just have to give up).
*/
type ErrWareCorrupt struct {
	Msg  string         `json:"msg"`
	Ware Ware           `json:"expected"`
	From WarehouseCoord `json:"from"`
}

func (e ErrWareCorrupt) Error() string {
	return fmt.Sprintf("Ware Corrupt: %s, while working on %q from %s", e.Msg, e.Ware.Hash, e.From)
}
