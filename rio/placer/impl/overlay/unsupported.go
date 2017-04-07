// +build !linux

package overlay

import (
	"go.polydawn.net/repeatr/rio"
)

func NewOverlayPlacer(workPath string) rio.Placer {
	panic("Overlay file system unsupported on this platform")
}
