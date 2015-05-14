package s3

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/input"
)

/*
	Raised if S3 credentials are not available.
*/
var S3CredentialsMissingError *errors.ErrorClass = input.ConfigError.NewClass("InputS3CredentialsMissingError")

/*
	Grouping for an error encountered while talking to the S3 API.
*/
var S3Error *errors.ErrorClass = input.Error.NewClass("InputS3Error")
