package placer

import (
	"polydawn.net/repeatr/io"
)

var _ integrity.Placer = CopyingPlacer

func CopyingPlacer(srcPath, destPath string, writable bool) integrity.Emplacement {
	return nil
}

var _ integrity.Placer = BindPlacer

func BindPlacer(srcPath, destPath string, writable bool) integrity.Emplacement {
	return nil
}

var _ integrity.Placer = AufsPlacer

func AufsPlacer(srcPath, destPath string, writable bool) integrity.Emplacement {
	return nil
}
