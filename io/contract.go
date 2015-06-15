package integrity

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
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
	Hash() CommitID
	Teardown()
}

type Transmat interface {
	/*
		`Materialize` causes data to exist, and returns an Arena which
		describes where the requested data can be seen on a local posix filesystem.

		In the bigger picture: Transmat.Materialize causes data to exist;
		Placers and Assemblers get several data sets to resemble a single
		filesystem; Executors then plop that whole thing into a sandbox.

		This may be a relatively long-running operation, and using a goroutine
		to parallelize materialize operations is advised.
		(See `executor/util/provision.go`.)

		Integrity: The Transmat implementation is responsible for ensuring that the content
		of this Arena matches the hash described by the `CommitID` parameter.

		Sharing and Mutability: The Transmat implementation is allowed to assume
		no other process will mutate the filesystem.  Thus, the Transmat may
		return an Arena that refers to a filesystem shared with other Arenas.
		**It is the caller's responsibility not to modify this filesystem, and undefined
		behavior may result from the whole system if this contract is violated.**
		The caller should use `io/placer` systems to isolate filesystems
		before operating on them or exposing them to other systems.

		Locality: Implementation-specific.  Many Transmats will actually produce
		data onto a plain local filesystem (and thus, behaviors of your host
		filesystem may leak through!  noatime, etc); others may use FUSE or
		other custom mounting sytems.
		The data need not all be available on the local machine by the time
		the Arena is returned; it may or may not be cached locally after Arena teardown.
		The behavior of fsync is not guaranteed.
		The only requirement is that it look roughly like a posix filesystem.

		Composability: Transmats may delegate based on the `TransmatKind`
		parameter.  See `integrity.DispatchingTransmat` for an example of this.

	*/
	Materialize(kind TransmatKind, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena

	Scan(kind TransmatKind, subjectPath string, siloURIs []SiloURI, options ...MaterializerConfigurer) CommitID

	/*
		Returns a list of all active Arenas managed by this Transmat.

		This isn't often used, since most work can be done through the idempotent
		materializer method, but it *is* critical for having the ability to do
		cleanup on a system that suffered an unexpected halt.
	*/
	//Arenas() []Arena
	// REMOVED: this doesn't seem to be very useful in general.  just for caching transmats.
	// subject to review, but a use case needs to be demonstrated, because this makes a bunch of things stateful that have no reason to be.
}

/*
	Factory function interface for Transmats.  Plugins must implement this.

	Takes a workdir path, and that's it.  This is may be expected to double-time
	it both as a fresh starter, and be able to recognize and attempt to
	recover ruins from a prior run.
*/
type TransmatFactory func(workPath string) Transmat

type MaterializerOptions struct {
	// TODO play more with how this pattern works (or doesn't) with embedding n stuff.
	// I'd be nice to have just one ProgressReporter configurator for both input and output systems, for example.

	ProgressReporter   chan<- float32
	AcceptHashMismatch bool
}

type MaterializerConfigurer func(*MaterializerOptions)

// not technically necessary as a type, but having this MaterializerFactoryConfigurer symbol exported means godoc groups things helpfully,

func ProgressReporter(rep chan<- float32) MaterializerConfigurer {
	return func(opts *MaterializerOptions) {
		opts.ProgressReporter = rep
	}
}

var AcceptHashMismatch = func(opts *MaterializerOptions) {
	opts.AcceptHashMismatch = true
}

func EvaluateConfig(options ...MaterializerConfigurer) MaterializerOptions {
	var opts MaterializerOptions
	for _, each := range options {
		each(&opts)
	}
	return opts
}

type Placer func(srcPath, destPath string, writable bool) Emplacement

type Emplacement interface {
	Teardown()
}

/*
	Assembles a filesystem from a bunch of scattered filesystem pieces.
	The source pieces will be rearranged into the single tree as fast as
	possible, and will not be modified.  (The fast systems use bind mounts
	and COW filesystems to isolate and rearrange; worst-case scenario,
	plain ol' byte copies get the job done.)
*/
type Assembler func(basePath string, mounts []AssemblyPart) Assembly

type Assembly interface {
	Teardown()
}

type AssemblyPart struct {
	TargetPath string // in the container fs context
	SourcePath string // datasource which we want to respect
	Writable   bool
}

// sortable by target path (which is effectively mountability order)
type AssemblyPartsByPath []AssemblyPart

func (a AssemblyPartsByPath) Len() int           { return len(a) }
func (a AssemblyPartsByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a AssemblyPartsByPath) Less(i, j int) bool { return a[i].TargetPath < a[j].TargetPath }

//
// coersion stuff
//

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

func NewDispatchingTransmat(workPath string, transmats map[TransmatKind]Transmat) *DispatchingTransmat {
	dt := &DispatchingTransmat{
		workPath: workPath,
		dispatch: transmats,
	}
	return dt
}

func (dt *DispatchingTransmat) Materialize(kind TransmatKind, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena {
	transmat := dt.dispatch[kind]
	if transmat == nil {
		panic(fmt.Errorf("no transmat of kind %q available to satisfy request", kind))
	}
	return transmat.Materialize(kind, dataHash, siloURIs, options...)
}

func (dt *DispatchingTransmat) Scan(kind TransmatKind, subjectPath string, siloURIs []SiloURI, options ...MaterializerConfigurer) CommitID {
	transmat := dt.dispatch[kind]
	if transmat == nil {
		panic(fmt.Errorf("no transmat of kind %q available to satisfy request", kind))
	}
	return transmat.Scan(kind, subjectPath, siloURIs, options...)
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

	This caching implementation assumes that everyone's working with plain
	directories, that we can move them, and that posix semantics fly.  In return,
	it's stateless and survives daemon reboots by pure coincidence with no
	additional persistence than the normal filesystem provides.

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
	err := os.MkdirAll(filepath.Join(ct.workPath, "committed"), 0755)
	if err != nil {
		panic(err) // TODO type.  migrate input/output errors
	}
	return ct
}

func (ct *CachingTransmat) Materialize(kind TransmatKind, dataHash CommitID, siloURIs []SiloURI, options ...MaterializerConfigurer) Arena {
	if dataHash == "" {
		// if you can't give us a hash, we can't cache.
		// also this is almost certainly doomed unless one of your options is `AcceptHashMismatch`, but that's not ours to check.
		return ct.DispatchingTransmat.Materialize(kind, dataHash, siloURIs, options...)
	}
	permPath := filepath.Join(ct.workPath, "committed", string(dataHash))
	_, statErr := os.Stat(permPath)
	if os.IsNotExist(statErr) {
		// TODO implement some terribly clever stateful parking mechanism, and do the real fetch in another routine.
		arena := ct.DispatchingTransmat.Materialize(kind, dataHash, siloURIs, options...)
		// keep it around.
		// build more realistic syncs around this later, but posix mv atomicity might actually do enough.
		err := os.Rename(arena.Path(), permPath)
		if err != nil {
			if err2, ok := err.(*os.LinkError); ok &&
				err2.Err == syscall.EBUSY || err2.Err == syscall.ENOTEMPTY {
				// oh, fine.  somebody raced us to it.
				if err := os.RemoveAll(arena.Path()); err != nil {
					panic(err) // not systemically fatal, but like, wtf mate.
				}
				return catchingTransmatArena{permPath}
			}
			panic(err)
		}
	}
	return catchingTransmatArena{permPath}
}

type catchingTransmatArena struct {
	path string
}

func (a catchingTransmatArena) Path() string   { return a.path }
func (a catchingTransmatArena) Hash() CommitID { return a.Hash() }
func (a catchingTransmatArena) Teardown()      { /* none */ }
