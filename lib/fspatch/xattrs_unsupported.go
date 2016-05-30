// +build !linux

package fspatch

import (
	"polydawn.net/repeatr/api/def"
)

func Lgetxattr(path string, attr string) ([]byte, error) {
	return nil, def.ErrNotSupportedPlatform
}

func Lsetxattr(path string, attr string, data []byte, flags int) error {
	return def.ErrNotSupportedPlatform
}
