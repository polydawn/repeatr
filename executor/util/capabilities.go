package util

import (
	"os"
	"path/filepath"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/placer"
	"polydawn.net/repeatr/io/transmat/dir"
	"polydawn.net/repeatr/io/transmat/tar"
	"polydawn.net/repeatr/io/transmat/tarexec"
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
		integrity.TransmatKind("dir"):      dirCacher,
		integrity.TransmatKind("tar"):      dirCacher,
		integrity.TransmatKind("exec-tar"): tarexec.New(filepath.Join(workDir, "tarexec")),
	})
	return universalTransmat
}

func BestAssembler() integrity.Assembler {
	if os.Getuid() != 0 {
		// Can't mount without root.
		return placer.NewAssembler(placer.CopyingPlacer)
	}
	if os.Getenv("TRAVIS") != "" {
		// Travis's own virtualization denies mounting.  whee.
		return placer.NewAssembler(placer.CopyingPlacer)
	}
	// If we *can* mount, AUFS+Bind is The Winner.
	return placer.NewAssembler(placer.NewAufsPlacer(filepath.Join(def.Base(), "aufs")))
	// TODO: fallbacks for mount but not aufs.
}
