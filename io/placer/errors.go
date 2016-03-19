package placer

import (
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/io"
)

var Error *errors.ErrorClass = rio.AssemblyError.NewClass("PlacerError")
