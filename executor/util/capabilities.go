package util

import (
	"path/filepath"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/dir"
	"polydawn.net/repeatr/io/tar"
)

/*
	The default, "universal", dispatching Transmat.
	You should be able to throw pretty much any type of input spec at it.

	If you're building your own transports and data warehousing integrations,
	you'll need to assemble your own Transmat instead of this one --
	`integrity.DispatchingTransmat` is good for composing them so you can still
	use one interface to get any kind of data you want.
*/
func DefaultTransmat() integrity.Transmat {
	workDir := filepath.Join(def.Base(), "io")
	dirCacher := integrity.NewCachingTransmat(filepath.Join(workDir, "dircacher"), map[integrity.TransmatKind]integrity.TransmatFactory{
		integrity.TransmatKind("dir"): dir.New,
		integrity.TransmatKind("tar"): tar.New,
	})
	universalTransmat := integrity.NewDispatchingTransmat(workDir, map[integrity.TransmatKind]integrity.Transmat{
		integrity.TransmatKind("dir"): dirCacher,
		integrity.TransmatKind("tar"): dirCacher,
	})
	return universalTransmat
}

// TODO the one we're *really* worried about is "pick the best assembler available"
