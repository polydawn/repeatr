package def

import (
	"github.com/spacemonkeygo/errors"
)

var ConfigError *errors.ErrorClass = errors.NewClass("ConfigError")

func newConfigValTypeError(expectedKey, mustBeA string, wasActually string) *errors.Error {
	return ConfigError.New("config key %q must be a %s; was %s", expectedKey, mustBeA, wasActually).(*errors.Error)
}
