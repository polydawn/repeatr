package s3

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/rio"
)

/*
	Raised if S3 credentials are not available.
*/
var S3CredentialsMissingError *errors.ErrorClass = rio.ConfigError.NewClass("InputS3CredentialsMissingError")
