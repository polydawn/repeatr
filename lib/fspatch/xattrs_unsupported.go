// +build !linux

package fspatch

func Lgetxattr(path string, attr string) ([]byte, error) {
	return nil, ErrUnsupportedPlatform
}

func Lsetxattr(path string, attr string, data []byte, flags int) error {
	return ErrUnsupportedPlatform
}
