package integrity

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"polydawn.net/repeatr/def"
)

/*
	dataHash - content address.  you know the drill.
	siloURI - a resource location where we can fetch from.
	workPath - a local filesystem path under which work must be done.
	  The work need only be *somewhere* under this tree (calls need to be able to
	  specify this in case they care about which mount data ends up on, etc).
	  The actual return path may be different.
*/
type Materializer func(dataHash string, siloURI string, workPath string) <-chan HaverReport

// so if we give up not having factories (see next comment chunk), workPath can probably move back to the factory.

// FIXME: this has forgetten that output might need a *setup* phase, and so is in fact stateful.
// ohhey, that's... why we had objects for this stuff since the beginning of time.
// womp.  i really wanted to get rid of a having a factory layer there if possible, but... guess that's just not possible.
type Slurper func(scanPath string, siloURI string) <-chan SlurpReport

type SlurpReport struct {
	Hash string
	Err  error
}

type HaverReport struct {
	Path string
	Err  error
}

type Placer func(srcPath, destPath string, writable bool) <-chan error

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

/*
	Creates temporary directories for the chained Materializer to operate in.
	This is a useful building block to make sure a Materializer can be used
	for multiple data sets without steping on its own toes.

	REVIEW: honestly, probably the least insane if every materializer quietly
	just does this internally.  Something that busts if called twice with
	the same workPath isn't really following the same method contract anyway.
*/
func TmpdirMaterializer(mat Materializer) Materializer {
	return func(dataHash string, siloURI string, workPath string) <-chan HaverReport {
		done := make(chan HaverReport)
		go func() {
			defer close(done)
			path, err := ioutil.TempDir(workPath, "")
			if err != nil {
				done <- HaverReport{Err: err}
				return
			}
			report := <-mat(dataHash, siloURI, path)
			if report.Err != nil {
				done <- report
				return
			}
			done <- HaverReport{Path: path}
		}()
		return done
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
	return func(dataHash string, siloURI string, workPath string) <-chan HaverReport {
		permPath := filepath.Join(workPath, "committed", dataHash)
		_, statErr := os.Stat(permPath)
		done := make(chan HaverReport)
		if os.IsNotExist(statErr) {
			stageBasePath := filepath.Join(workPath, "staging")
			go func() {
				defer close(done)
				report := <-TmpdirMaterializer(mat)(dataHash, siloURI, stageBasePath)
				// keep it around.
				// build more realistic syncs around this later, but posix mv atomicity might actually do enough.
				err := os.Rename(report.Path, permPath)
				if err != nil {
					done <- HaverReport{Err: err}
					return
				}
				done <- HaverReport{Path: permPath}
			}()
		} else {
			done <- HaverReport{Path: permPath}
			close(done)
		}
		return done
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
	filesystems := make([]HaverReport, len(formula.Inputs))
	var inputWg sync.WaitGroup
	for i, input := range formula.Inputs {
		inputWg.Add(1)
		go func() {
			filesystems[i] = <-materializer(input.Hash, input.URI, workDir)
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
