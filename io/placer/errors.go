package placer

import (
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/io"
)

var Error *errors.ErrorClass = integrity.AssemblyError.NewClass("PlacerError")
