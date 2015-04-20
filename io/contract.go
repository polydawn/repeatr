package integrity

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"polydawn.net/repeatr/def"
)

type SiloURI string

type CommitID string

type Arena interface {
	Path() string
	Archive(siloURIs []SiloURI) CommitID
	Teardown()
}

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

type Materializer func(workPath string, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena

//type Slurper func(scanPath string, siloURI string) <-chan SlurpReport
// GONE as a concept.  Any data installation can now be scanned, and output arenas are just denoted by the magic zero CommitID.
// Well, maybe not quite that much magic value on CommitIDs.  That might be poor.

type Placer func(srcPath, destPath string, writable bool)

/*
	Writable inputs get a COW.
	RO inputs just bind.
	Outputs (always writable) don't have input, so also can just bind.

	Expect most assemblers to be constructed with a Haver and a Placer.
*/
type Assembler func([]AssemblyPart)

type AssemblyPart struct {
	TargetPath string // in the container fs context
	SourcePath string // datasource which we want to respect
	Writable   bool
	// TODO make sure we get an example that sees how this reacts to outputs: not sure we have enough bits here yet
}

// sortable by target path (which is effectively mountability order)
type Assembly []AssemblyPart

func (a Assembly) Len() int           { return len(a) }
func (a Assembly) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Assembly) Less(i, j int) bool { return a[i].TargetPath < a[j].TargetPath }

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

/*
	Creates temporary directories for the chained Materializer to operate in.
	This is a useful building block to make sure a Materializer can be used
	for multiple data sets without steping on its own toes.

	REVIEW: honestly, probably the least insane if every materializer quietly
	just does this internally.  Something that busts if called twice with
	the same workPath isn't really following the same method contract anyway.
*/
func TmpdirMaterializer(mat Materializer) Materializer {
	return func(workPath string, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena {
		path, err := ioutil.TempDir(workPath, "")
		if err != nil {
			panic(err)
		}
		arena := mat(path, dataHash, siloURIs, options...)
		// remove tempdir on your way out
		return &teardownDelegatingArena{Delegate: arena, TeardownFn: func() { os.RemoveAll(path) }}
	}
}

/*
	Proxies a Materializer, keeping a cache of filesystems that are requested.
	The cache is based purely on dataHash (it does not have any knowledge of
	the innards of other Materializers).

	Filesystems returned presumed *not* be modified, or behavior is undefined and the
	cache becomes unsafe to use.  Use should be combined with some kind of `Placer`
	that preserves integrity of the cached filesystem.
*/
func CachingMaterializer(mat Materializer) Materializer {
	return func(workPath string, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena {
		permPath := filepath.Join(workPath, "committed", string(dataHash))
		_, statErr := os.Stat(permPath)
		if os.IsNotExist(statErr) {
			stageBasePath := filepath.Join(workPath, "staging")
			// TODO implement some terribly clever stateful parking mechanism, and do the real fetch in another routine.
			arena := TmpdirMaterializer(mat)(stageBasePath, dataHash, siloURIs, options...)
			// keep it around.
			// build more realistic syncs around this later, but posix mv atomicity might actually do enough.
			err := os.Rename(arena.Path(), permPath)
			if err != nil {
				panic(err)
			}
			// TODO you should have another... what, locationProxyingArena here?
			// ponder what this implies about the sanity level of the Arena interface!
			// can these generally *be* moved?  is that a thing??  **probably not** with the mounty ones!
			// ... well that certainly got tricky.
			// i guess we're tipping back towards imperative styles again, then.
			return arena
		} else {
			return nil
			// TODO so... again, does this Arena interface make sense?
			// this would... *not* run teardown commands, presumably?
			// I think attaching commands to this Arena interface is not going to go well, due
			// to the fact that we have to have sanifying cleanups be possible after nothing less than machine hard-downs.
			// Which means we need to recover arena descriptions from serial, fsync'd data.
			// And be able to use that to tell a driver to Do Things like teardown.
			// I guess really just teardown is about it, but that's still a pretty unignorable case, and thereafter we might as well be consistent with using a driver pattern.
			// Maybe it's still workable to have an Arena interface, as long as there's a reasonable `(d *Driver) Recover(messyPath string) Arena` interface and some kind of recognizer dispatch system.
		}
	}
}

type TheAssembler struct {
	Placer Placer
}

var _ Assembler = (&TheAssembler{}).Assemble

func (a *TheAssembler) Assemble(mounts []AssemblyPart) {
	sort.Sort(Assembly(mounts))
	//	for
}

//
// exapmle
//

func example() {
	var formula def.Formula
	var materializer Materializer // this is ignoring dispatcher for now
	var wantCache bool
	var workDir string // probably one per executor; whatever

	// materializers have a consistent interface so we can drop cachers in or out, transparently.
	if wantCache {
		materializer = CachingMaterializer(materializer)
		workDir = filepath.Join(workDir, "cache")
	} else {
		materializer = TmpdirMaterializer(materializer)
		workDir = filepath.Join(workDir, "tmp")
	}

	// start having all filesystems
	// large amounts of this would maybe make sense to get DRY and shoved in the assembler
	filesystems := make([]Arena, len(formula.Inputs))
	var inputWg sync.WaitGroup
	for i, input := range formula.Inputs {
		inputWg.Add(1)
		go func() {
			filesystems[i] = materializer(workDir, CommitID(input.Hash), []SiloURI{SiloURI(input.URI)})
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
