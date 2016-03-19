package assets

import (
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/rio"
)

var ErrLoadingAsset *errors.ErrorClass = rio.Error.NewClass("ErrLoadingAsset")
