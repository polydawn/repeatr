package gs

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/io"
)

/*
	Raised if GS credentials are not available.
*/
var GsCredentialsMissingError *errors.ErrorClass = integrity.ConfigError.NewClass("InputGsCredentialsMissingError")
var GsCredentialsInvalidError *errors.ErrorClass = integrity.ConfigError.NewClass("InputGsCredentialsInvalid")
