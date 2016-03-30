package placer

import (
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/rio"
)

var Error *errors.ErrorClass = rio.AssemblyError.NewClass("PlacerError")
