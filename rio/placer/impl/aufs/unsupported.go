// +build !linux

package aufs

import (
	"go.polydawn.net/repeatr/rio"
)

func NewAufsPlacer(workPath string) rio.Placer {
	panic("AufsPlacer unsupported on this platform")
}
