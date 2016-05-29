package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/polydawn/gosh"

	"polydawn.net/repeatr/core/jank"
	"polydawn.net/repeatr/rio"
	"polydawn.net/repeatr/rio/placer"
	"polydawn.net/repeatr/rio/transmat/impl/cachedir"
	"polydawn.net/repeatr/rio/transmat/impl/dir"
	"polydawn.net/repeatr/rio/transmat/impl/git"
	"polydawn.net/repeatr/rio/transmat/impl/s3"
	"polydawn.net/repeatr/rio/transmat/impl/tar"
)

/*
	The default, "universal", dispatching Transmat.
	You should be able to throw pretty much any type of input spec at it.

	If you're building your own transports and data warehousing integrations,
	you'll need to assemble your own Transmat instead of this one --
	`rio.DispatchingTransmat` is good for composing them so you can still
	use one interface to get any kind of data you want.
*/
func DefaultTransmat() rio.Transmat {
	workDir := filepath.Join(jank.Base(), "io")
	dirCacher := cachedir.New(filepath.Join(workDir, "dircacher"), map[rio.TransmatKind]rio.TransmatFactory{
		rio.TransmatKind("dir"): dir.New,
		rio.TransmatKind("tar"): tar.New,
		rio.TransmatKind("s3"):  s3.New,
	})
	gitCacher := cachedir.New(filepath.Join(workDir, "dircacher-git"), map[rio.TransmatKind]rio.TransmatFactory{
		rio.TransmatKind("git"): git.New,
	})
	universalTransmat := rio.NewDispatchingTransmat(map[rio.TransmatKind]rio.Transmat{
		rio.TransmatKind("dir"): dirCacher,
		rio.TransmatKind("tar"): dirCacher,
		rio.TransmatKind("s3"):  dirCacher,
		rio.TransmatKind("git"): gitCacher,
	})
	return universalTransmat
}

func BestAssembler() rio.Assembler {
	if bestAssembler == nil {
		bestAssembler = determineBestAssembler()
	}
	return bestAssembler
}

var bestAssembler rio.Assembler

func determineBestAssembler() rio.Assembler {
	if os.Getuid() != 0 {
		// Can't mount without root.
		fmt.Fprintf(os.Stderr, "WARN: using slow fs assembly system: need root privs to use faster systems.\n")
		return placer.NewAssembler(placer.CopyingPlacer)
	}
	if os.Getenv("TRAVIS") != "" {
		// Travis's own virtualization denies mounting.  whee.
		fmt.Fprintf(os.Stderr, "WARN: using slow fs assembly system: travis' environment blocks faster systems.\n")
		return placer.NewAssembler(placer.CopyingPlacer)
	}
	// If we *can* mount...
	if isAUFSAvailable() {
		// if AUFS is installed, AUFS+Bind is The Winner.
		return placer.NewAssembler(placer.NewAufsPlacer(filepath.Join(jank.Base(), "aufs")))
	}
	// last fallback... :( copy it is
	fmt.Fprintf(os.Stderr, "WARN: using slow fs assembly system: install AUFS to use faster systems.\n")
	return placer.NewAssembler(placer.CopyingPlacer)
	// TODO we should be able to use copy for fallback RW isolator but still bind for RO.  write a new placer for that.  or really, maybe bind should chain.
}

func isAUFSAvailable() bool {
	// the greatest thing to do would of course just be to issue the syscall once and see if it flies
	// but that's a distrubingly stateful and messy operation so we're gonna check a bunch
	// of next-best-things instead.

	// If we it's in /proc/filesystems, we should be good to go.
	// (If it's not, the libs might be installed, but not loaded, so we'll try that.)
	if fs, err := ioutil.ReadFile("/proc/filesystems"); err == nil {
		fsLines := strings.Split(string(fs), "\n")
		for _, line := range fsLines {
			parts := strings.Split(line, "\t")
			if len(parts) < 2 {
				continue
			}
			if parts[1] == "aufs" {
				return true
			}
		}
	}

	// Blindly attempt to modprobe the AUFS module into the kernel.
	// If it works, great.  If it doesn't, okay, we'll move on.
	// Repeatedly installing it if it already exists no-op's correctly.
	// Timeout is 100ms... maybe a little aggressive, but this takes 36ms on
	//  my machine with a cold disk cache, 11ms hot, and is remarkably consistent.
	modprobeCode := gosh.Sh(
		"modprobe", "aufs",
		gosh.NullIO,
		gosh.Opts{OkExit: gosh.AnyExit},
	).GetExitCodeSoon(100 * time.Millisecond)
	if modprobeCode != 0 {
		return false
	}

	return true
}
