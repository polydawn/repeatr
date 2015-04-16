package integrity

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"polydawn.net/repeatr/def"
)

type Materializer func(dataHash string, siloURI string, destPath string) <-chan error

type Slurper func(scanPath string) <-chan SlurpReport

type SlurpReport struct {
	Hash string
	Err  error
}

type Haver func(dataHash string, siloURI string) <-chan HaverReport // so... yes, we're almost certainly willing to compromize this to have a destPath param that may or may not be blatantly ignored.

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

func TmpdirHaver(workPath string, mat Materializer) Haver {
	return func(dataHash string, siloURI string) <-chan HaverReport {
		done := make(chan HaverReport)
		go func() {
			defer close(done)
			path, err := ioutil.TempDir(workPath, "")
			if err != nil {
				done <- HaverReport{Err: err}
				return
			}
			err = <-mat(dataHash, siloURI, path)
			if err != nil {
				done <- HaverReport{Err: err}
				return
			}
			done <- HaverReport{Path: path}
		}()
		return done
	}
}

func CachingHaver(workPath string, mat Materializer) Haver {
	return func(dataHash string, siloURI string) <-chan HaverReport {
		permPath := filepath.Join(workPath, "committed", dataHash)
		_, statErr := os.Stat(permPath)
		done := make(chan HaverReport)
		if os.IsNotExist(statErr) {
			stageBasePath := filepath.Join(workPath, "staging")
			go func() {
				defer close(done)
				report := <-TmpdirHaver(stageBasePath, mat)(dataHash, siloURI)
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

	// we start with materializers but always coerce them into acting like havers,
	// just so we can have a consistent interface and drop the caching layer transparently.
	var haver Haver

	// materializers should be draftable into havers... with or without cachers
	if wantCache {
		haver = CachingHaver(filepath.Join(workDir, "cache"), materializer)
	} else {
		haver = TmpdirHaver(filepath.Join(workDir, "tmp"), materializer)
	}

	// start having all filesystems
	//var []HaverReport
	for _, input := range formula.Inputs {
		//func(dataHash string, siloURI string) <-chan HaverReport
		// do a for loop you fool
		haver(input.Hash, input.URI)
		// TODO collect
	}
	for _, output := range formula.Outputs {
		_ = output
	}

	// assemble them into the final tree
}
