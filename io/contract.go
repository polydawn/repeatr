package integrity

import (
	"path/filepath"
	"polydawn.net/repeatr/def"
	tarinput "polydawn.net/repeatr/input/tar2"
)

// integrity
// faith

// tricky bits:
//   - disbatch happens in multiple layers.
//     - there's a basic one, and there's another one needed the instant you introduce the cache layer.
//     - prepare the plugin system for this!

// siloType, dataHash, siloLocation, localLocation >-- MATERIALIZER --> chan {}

// siloType, localLocation, siloLocation >-- SLURPER --> chan dataHash

// The major change here is that it's deeply important that the materializer must *yield* a localLocation instead of accept a localLocation:
//  this makes it possible to have a trusting cacher do the same manuver.
// Putting something into a final location is always a separate step:
//  Usually that's going to take the place of a bind mount (and may involve an indirection that does COW).
//  It *might* take

// FIXME: no, still dumb.  The *bottom* of the universe still has to be an imperative localLocation.

// Yeah, give the cache a placer.
// The cache can internally plan to share (or be exclusive on) a filesystem area,
//  and no part of this impacts executors setting up their own cachers with COW semantics appropriate for them.

// ... What about an executor that just wants the truth and no action?
//  Sorry, impossible.  Or, the cacher can give you an interface for that too,
//   but most consumers are going to use the `func (c *cacher) Materializer(Placer) func(siloType, dataHash, siloLocation, localLocation) <-chan err` route.
//    Hell, that can be a `func AdaptMaterializer(Cacher, Placer) Materializer`.  Nothing's likely to change per cacher.

//
//type Materializer()

// LET'S TRY THIS AGAIN.

// dataHash, localLocation >-- MATERIALIZER{siloType,[]satisfiers} --> chan {}

// localLocation >-- SLURPER{siloType,[]archiveDestinations} --> chan dataHash

// oh, hey.  look: the contents of the middle shit are actually the same shape.

type Materializer func(dataHash string, siloURI string, destPath string, writable bool) <-chan error

type Slurper func(scanPath string) <-chan SlurpReport

type SlurpReport struct {
	Hash string
	Err  error
}

type Placer func(srcPath, destPath string, writable bool) <-chan error

// n.b. dat errur handlin goonna be odd: placer errors, are they input or output?  wrap ur placer for conversion?  embarass.  but, work.

func TarMaterializer() Materializer { // this doesn't need to be a factory anymore :/  all the parameters evaporated
	return func(dataHash string, siloURI string, destPath string, _ bool) <-chan error {
		return tarinput.New(def.Input{
			Type: "tar",
			Hash: dataHash,
			URI:  siloURI,
		}).Apply(destPath)
	}
}

func AufsPlacer(workDirPath string) Placer {
	return func(srcPath, destPath string, _ bool) <-chan error {
		// use workdir to host a tempdir to use as the "layer" dir
		return nil // ignore writable, COW it -- use in combo with bindmounter that understands ro
	}
}

func BindPlacer(writerIsolator Placer) Placer {
	return func(srcPath, destPath string, writable bool) <-chan error {
		if writable {
			return writerIsolator(srcPath, destPath, writable)
			// no need to do another bounce of bind; we're already mounting so we can just do it in place
		} else {
			// plain bind mount with ro bits
			return nil
		}
	}

}

func AdaptMaterializer(c *Cacher, p Placer) Materializer {
	return func(dataHash string, siloURI string, destPath string, writable bool) <-chan error {
		cachePath := c.Get(dataHash, siloURI)
		return p(cachePath, destPath, writable)
	}
}

type Cacher struct {
	BasePath string       // working dir
	filler   Materializer // NOTE you lost the ability to have another layer of disbatcher here, which might be handy if two different types of materializer happen to safely share a hash space (which, jussayin, several *do*)... though, no, that's sane since the get call just accepts a siloLocation... for now...
}

func (c *Cacher) Get(dataHash string, siloURI string) (path string) { // blocks
	cachePath := filepath.Join(c.BasePath, dataHash)
	// todo something with claims and joining waits too
	fillErr := <-c.filler(dataHash, siloURI, cachePath, false)
	panic(fillErr)
	return cachePath
}

func example() {
	// assemble a caching tar input system, all the way from job config stuff.
	inSpec := def.Input{
		Type: "tar",
		Hash: "2lkf8vsd",
		URI:  "file://data/supplier",
	}

	matDispatcher := MaterializerDispatcher{
		"tar": TarMaterializer(),
	}

	var mat Materializer = matDispatcher[inSpec.Type]

	mat(inSpec.Hash, inSpec.URI, inSpec.Location /* translated for rootfs, obvs */, true)

	_ = mat
}

type MaterializerDispatcher map[string]Materializer

// guess what else
// guess what else

// the last stage of placer needs to be in a different level of sync than anything else
// because the executor might need to do some mkdirs in between the last couple placer operations
// and it most definitely does not want to have a wildly variable and networked wait inbetween thoseS

// bonus points: please remember that this is going to need progress reporting.
// anyone returning an error chan right now should be returning a progress monitor instead.
