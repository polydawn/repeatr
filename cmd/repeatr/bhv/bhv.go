package cmdbhv

import (
	"go.polydawn.net/meep"
)

type ErrBadArgs struct {
	meep.TraitAutodescribing
	Message string
}
