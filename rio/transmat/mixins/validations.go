package mixins

import (
	"fmt"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/rio"
)

// Raises `*rio.ErrInternal` if the kinds don't match up.
func MustBeType(must, actual rio.TransmatKind) {
	if must != actual {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Incorrect dispatch: " +
				fmt.Sprintf("supports definitions of type %q, not %q", must, actual)},
		))
	}
}
