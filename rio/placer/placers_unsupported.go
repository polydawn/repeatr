// +build !linux
package placer

import (
	"go.polydawn.net/repeatr/rio"
)

var _ rio.Placer = BindPlacer

func BindPlacer(srcPath, destPath string, writable bool, _ bool) rio.Emplacement {
	panic("BindPlacer unsupported on darwin")
}

func NewAufsPlacer(workPath string) rio.Placer {
	panic("AufsPlacer unsupported on darwin")
}
