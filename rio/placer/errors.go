package placer

import (
	"github.com/spacemonkeygo/errors"

	"go.polydawn.net/repeatr/rio"
)

var Error *errors.ErrorClass = rio.AssemblyError.NewClass("PlacerError")
