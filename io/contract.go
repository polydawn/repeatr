package integrity

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"polydawn.net/repeatr/def"
)

/*
	String describing a type of data transmat.  These are the keys used in plugin registration,
	and are used to dispatch input/output configurations to their appropriate drivers.

	TransmatKind labels must be devoid of slashes and other special characters.
*/
type TransmatKind string

type SiloURI string

type CommitID string

type Arena interface {
	Path() string
	Archive(siloURIs []SiloURI) CommitID
	Teardown()
}

// You *need* some Transmat interface to gather these.
// You *might* get some use out of having them as free-floating functional interfaces, but it's frankly not clear.
// This has a factory interface with a workdir, and that's it: it's expected to double-time it as a recovery recognizer and a fresh starter.
// We're not gonna do a recognizer disbatcher because I can't think of a legitmate situation where we'd be
//   - barreling into a filesystem like that with no preexisting expectations
//   - and reasonably be able to reuse anything there.
// Cleanup of things we *don't* recognize should, I think, be pretty consistently a series of umount and rm's; I don't know of any exceptions to that, and these can be done without knowing what set the stuff up.
//
// Patterns of use:
//   - Any time you see a dispatcher, it's going to be talking about transmats.
//   - Any time you're *building* a dispatcher, you're going to be talking about transmat factories -- you invoke them as you're setting up the dispatcher, and also you might be chaining them into each other.
//
type Transmat interface {
	Materialize(kind TransmatKind, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena

	/*
		Returns a list of all active Arenas managed by this Transmat.

		This isn't often used, since most work can be done through the idempotent
		materializer method, but it *is* critical for having the ability to do
		cleanup on a system that suffered an unexpected halt.
	*/
	Arenas() []Arena
}

type TransmatFactory func(workPath string) Transmat

type MaterializerOptions struct {
	// TODO play more with how this pattern works (or doesn't) with embedding n stuff.
	// I'd be nice to have just one ProgressReporter configurator for both input and output systems, for example.
	// TODO probably also needs exported symbols so any third party systems can read the config too!

	progressReporter chan<- float32
}

type MaterializerConfigurer func(*MaterializerOptions)

// not technically necessary as a type, but having this MaterializerFactoryConfigurer symbol exported means godoc groups things helpfully,

func ProgressReporter(rep chan<- float32) MaterializerConfigurer {
	return func(opts *MaterializerOptions) {
		opts.progressReporter = rep
	}
}

//type Slurper func(scanPath string, siloURI string) <-chan SlurpReport
// GONE as a concept.  Any data installation can now be scanned, and output arenas are just denoted by the magic zero CommitID.
// Well, maybe not quite that much magic value on CommitIDs.  That might be poor.

type Placer func(srcPath, destPath string, writable bool) Emplacement

type Emplacement interface {
	Teardown()
}

/*
	Writable inputs get a COW.
	RO inputs just bind.
	Outputs (always writable) don't have input, so also can just bind.

	Expect most assemblers to be constructed with a Haver and a Placer.
*/
type Assembler func(basePath string, mounts []AssemblyPart) Assembly

type Assembly interface {
	Teardown()
}

type AssemblyPart struct {
	TargetPath string // in the container fs context
	SourcePath string // datasource which we want to respect
	Writable   bool
	// TODO make sure we get an example that sees how this reacts to outputs: not sure we have enough bits here yet
	//  ... indeed, dealing with outputs does rather make it clear that a copying placer isn't an acceptable drop-in mechanism.
}

// sortable by target path (which is effectively mountability order)
type AssemblyPartsByPath []AssemblyPart

func (a AssemblyPartsByPath) Len() int           { return len(a) }
func (a AssemblyPartsByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a AssemblyPartsByPath) Less(i, j int) bool { return a[i].TargetPath < a[j].TargetPath }

//
// coersion stuff
//

var _ Arena = &teardownDelegatingArena{}

type teardownDelegatingArena struct {
	Delegate   Arena
	TeardownFn func()
}

func (a *teardownDelegatingArena) Path() string { return a.Delegate.Path() }
func (a *teardownDelegatingArena) Archive(siloURIs []SiloURI) CommitID {
	return a.Delegate.Archive(siloURIs)
}
func (a *teardownDelegatingArena) Teardown() { defer a.TeardownFn(); a.Delegate.Teardown() }

var _ Transmat = &DispatchingTransmat{}

/*
	DispatchingTransmat gathers a bunch of Transmats under one entrypoint,
	so that any kind of data specification can be fed into this one `Materialize`
	function, and it will DTRT.
*/
type DispatchingTransmat struct {
	workPath string
	dispatch map[TransmatKind]Transmat
}

func NewDispatchingTransmat(workPath string, transmats map[TransmatKind]TransmatFactory) *DispatchingTransmat {
	dt := &DispatchingTransmat{
		workPath: workPath,
		dispatch: make(map[TransmatKind]Transmat, len(transmats)),
	}
	for kind, factoryFn := range transmats {
		dt.dispatch[kind] = factoryFn(filepath.Join(workPath, "stg", string(kind)))
	}
	return dt
}

func (dt *DispatchingTransmat) Arenas() []Arena {
	var a []Arena
	for _, transmat := range dt.dispatch {
		a = append(a, transmat.Arenas()...)
	}
	return a
}

func (dt *DispatchingTransmat) Materialize(kind TransmatKind, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena {
	transmat := dt.dispatch[kind]
	if transmat == nil {
		panic(fmt.Errorf("no transmat of kind %q available to satisfy request", kind))
	}
	return transmat.Materialize(kind, dataHash, siloURIs, options...)
}

var _ Transmat = &CachingTransmat{}

/*
	Proxies a Transmat (or set of dispatchable Transmats), keeping a cache of
	filesystems that are requested.

	Caching is based on CommitID.  Thus, any repeated requests for the same CommitID
	can be satisfied instantly, and this system does not have any knowledge of
	the innards of other Transmat, so it can be used with any valid Transmat.
	(Obviously, this also means this will *not* help any two Transmats magically
	do dedup on data *within* themselves at a higher resolution than full dataset
	commits, by virtue of not having that much understanding of proxied Transmats.)
	If two different Transmats happen to share the same CommitID "space" ("dir" and "tar"
	systems do, for example), then they may share a CachingTransmat; constructing
	a CachingTransmat that proxies more than one Transmat that *doesn't* share the same "space"
	is undefined and unwise.

	Filesystems returned are presumed *not* to be modified, or behavior is undefined and the
	cache becomes unsafe to use.  Use should be combined with some kind of `Placer`
	that preserves integrity of the cached filesystem.
*/
type CachingTransmat struct {
	DispatchingTransmat
}

func NewCachingTransmat(workPath string, transmats map[TransmatKind]TransmatFactory) *CachingTransmat {
	// Note that this *could* be massaged to fit the TransmatFactory signiture, but there doesn't
	//  seem to be a compelling reason to do so; there's not really any circumstance where
	//  you'd want to put a caching factory into a TransmatFactory registry as if it was a plugin.
	ct := &CachingTransmat{
		DispatchingTransmat{
			workPath: workPath,
			dispatch: make(map[TransmatKind]Transmat, len(transmats)),
		},
	}
	for kind, factoryFn := range transmats {
		ct.dispatch[kind] = factoryFn(filepath.Join(workPath, "stg", string(kind)))
	}
	return ct
}

func (ct *CachingTransmat) Materialize(kind TransmatKind, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena {
	permPath := filepath.Join(ct.workPath, "committed", string(dataHash))
	// TODO everything about this prototype that mentions os.Stat and os.Rename needs to be replaced.
	// We can't use the filesystem as the primary data storage; we can't do the rename trick for
	// all possibile systems, so we're going to need our own state tracking system.
	_, statErr := os.Stat(permPath)
	if os.IsNotExist(statErr) {
		// TODO implement some terribly clever stateful parking mechanism, and do the real fetch in another routine.
		arena := ct.DispatchingTransmat.Materialize(kind, dataHash, siloURIs, options...)
		// keep it around.
		// build more realistic syncs around this later, but posix mv atomicity might actually do enough.
		err := os.Rename(arena.Path(), permPath)
		if err != nil {
			panic(err)
		}
		return arena
	} else {
		return nil // TODO return existing (which we should already have proxied ref to that has a noop teardown)
	}
}

//
// exapmle
//

func example() {
	var formula def.Formula
	var workDir string // probably one per executor; whatever

	// pretend we have a bunch of diverse transmat systems implemented.
	// these'll be things we have as registerable pluginnable systems.
	var dirTransmat TransmatFactory
	var tarTransmat TransmatFactory
	var ipfsTransmat TransmatFactory

	// transmats have a consistent interface so we can drop cachers in or out, transparently.
	// and we can assemble dispatchers covering the whole spectrum.
	dirCacher := NewCachingTransmat(filepath.Join(workDir, "dircacher"), map[TransmatKind]TransmatFactory{
		TransmatKind("dir"): dirTransmat,
		TransmatKind("tar"): tarTransmat,
	})
	universalTransmat := NewDispatchingTransmat(workDir, map[TransmatKind]TransmatFactory{
		TransmatKind("dir"):  func(_ string) Transmat { return dirCacher }, // REVIEW this seems odd; maybe these things shouldn't take factories at all.
		TransmatKind("tar"):  func(_ string) Transmat { return dirCacher },
		TransmatKind("ipfs"): ipfsTransmat,
	})

	// start having all filesystems
	// large amounts of this would maybe make sense to get DRY and shoved in the assembler
	filesystems := make([]Arena, len(formula.Inputs))
	var inputWg sync.WaitGroup
	for i, input := range formula.Inputs {
		inputWg.Add(1)
		go func() {
			filesystems[i] = universalTransmat.Materialize(TransmatKind(input.Type), CommitID(input.Hash), []SiloURI{SiloURI(input.URI)})
			// TODO these are now synchronous and emit errors here; need try block
			// could: do something clever with errors here instant emit cancels to everything else.
			inputWg.Done()
		}()
	}
	for _, output := range formula.Outputs {
		_ = output
		// TODO output setups
	}
	inputWg.Wait()

	// assemble them into the final tree
	// TODO

	// "run something", if this were a real executor

	// run commit on the outputs
	// TODO
}
