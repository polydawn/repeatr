// +build !linux

package bind

import (
	"go.polydawn.net/repeatr/rio"
)

func BindPlacer(srcPath, destPath string, writable bool, _ bool) rio.Emplacement {
	panic("BindPlacer unsupported on this platform")
}
