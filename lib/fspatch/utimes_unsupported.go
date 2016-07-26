// +build !linux,!freebsd,!darwin

package fspatch

import (
	"time"

	"go.polydawn.net/repeatr/api/def"
)

func LUtimesNano(path string, atime time.Time, mtime time.Time) error {
	return def.ErrNotSupportedPlatform
}

func UtimesNano(path string, atime time.Time, mtime time.Time) error {
	return def.ErrNotSupportedPlatform
}
