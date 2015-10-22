package assets

import (
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/io"
)

var ErrLoadingAsset *errors.ErrorClass = integrity.Error.NewClass("ErrLoadingAsset")
