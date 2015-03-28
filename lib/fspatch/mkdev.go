package fspatch

// Workaround for https://github.com/golang/go/issues/8106 .
// Remove freely after a fix for that lands upstream.
func Mkdev(major int64, minor int64) uint32 {
	return uint32(((minor & 0xfff00) << 12) | ((major & 0xfff) << 8) | (minor & 0xff))
}
