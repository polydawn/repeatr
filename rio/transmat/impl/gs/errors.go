package gs

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/rio"
)

/*
	Raised if GS credentials are not available.
*/
var GsCredentialsMissingError *errors.ErrorClass = rio.ConfigError.NewClass("InputGsCredentialsMissingError")
var GsCredentialsInvalidError *errors.ErrorClass = rio.ConfigError.NewClass("InputGsCredentialsInvalid")
