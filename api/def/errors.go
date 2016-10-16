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
