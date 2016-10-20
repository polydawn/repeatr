package s3

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/rlmcpherson/s3gof3r"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/lib/fshash"
	"go.polydawn.net/repeatr/rio"
	tartrans "go.polydawn.net/repeatr/rio/transmat/impl/tar"
	"go.polydawn.net/repeatr/rio/transmat/mixins"
)

const Kind = rio.TransmatKind("s3")

var _ rio.Transmat = &S3Transmat{}

type S3Transmat struct {
	workPath string
}

var _ rio.TransmatFactory = New

func New(workPath string) rio.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Unable to set up workspace"},
			meep.Cause(err),
		))
	}
	return &S3Transmat{workPath}
}

var hasherFactory = sha512.New384

/*
	Arenas produced by Dir Transmats may be relocated by simple `mv`.
*/
func (t *S3Transmat) Materialize(
	kind rio.TransmatKind,
	dataHash rio.CommitID,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.Arena {
	var arena dirArena
	meep.Try(func() {
		// Basic validation and config
		mixins.MustBeType(Kind, kind)
		config := rio.EvaluateConfig(options...)

		// Ping silos
		if len(siloURIs) < 1 {
			// Note that it's possible a caching layer will satisfy things even without data sources...
			//  but if that was going to happen, it already would have by now.
			panic(&def.ErrWarehouseUnavailable{
				Msg:    "No warehouse coords configured!",
				During: "fetch",
			})
		}
		// Load keys.
		keys := mustLoadKeys()
		// Our policy is to take the first path that exists.
		//  This lets you specify a series of potential locations,
		//   and if one is unavailable we'll just take the next.
		var warehouse *Warehouse
		var anyoneOutThere bool
		for _, uri := range siloURIs {
			wh := NewWarehouse(uri, keys)
			err := wh.PingReadable()
			if err == nil {
				anyoneOutThere = true
				warehouse = wh
				break
			}
			meep.TryPlan{
				{ByType: &def.ErrWarehouseUnavailable{}, Handler: func(_ error) {
					// fine, we'll just try the next one
					log.Info("Warehouse not available, skipping", "warehouse", uri)
				}},
				{ByType: &def.ErrWareDNE{}, Handler: func(_ error) {
					// fine, we'll just try the next one
					anyoneOutThere = true // but at least someone was *alive*
					log.Info("Warehouse does not have the data, skipping", "warehouse", uri, "hash", dataHash)
				}},
			}.MustHandle(err)
		}
		if !anyoneOutThere {
			panic(&def.ErrWarehouseUnavailable{
				Msg:    "No warehouses responded!",
				During: "fetch",
			})
		}
		if warehouse == nil {
			panic(&def.ErrWareDNE{
				Ware: def.Ware{Type: string(Kind), Hash: string(dataHash)},
			})
		}

		// initialize reader from s3!
		stream := warehouse.openReader(dataHash)
		defer stream.Close()

		// Wrap input stream with decompression as necessary
		reader, err := tartrans.Decompress(stream)
		if err != nil {
			panic(&def.ErrWareCorrupt{
				Msg:  fmt.Sprintf("could not start decompressing: %s", err),
				Ware: def.Ware{Type: string(Kind), Hash: string(dataHash)},
				From: warehouse.coord,
			})
		}
		tarReader := tar.NewReader(reader)

		// Create staging arena to produce data into.
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(meep.Meep(
				&rio.ErrInternal{Msg: "Unable to create arena"},
				meep.Cause(err),
			))
		}

		// walk input tar stream, placing data and accumulating hashes and metadata for integrity check
		bucket := &fshash.MemoryBucket{}
		tartrans.Extract(tarReader, arena.Path(), bucket, hasherFactory, log)

		// bucket processing may have created a root node if missing.  if so, we need to apply its props.
		fs.PlaceFile(arena.Path(), bucket.Root().Metadata, nil)

		// hash whole tree
		actualTreeHash := base64.URLEncoding.EncodeToString(fshash.Hash(bucket, hasherFactory))

		// verify total integrity
		expectedTreeHash := string(dataHash)
		// If we got what we asked for: excellent, return.
		if actualTreeHash == expectedTreeHash {
			arena.hash = dataHash
			return
		}
		// If not... this may or may not be grounds for panic, depending on configuration.
		if config.AcceptHashMismatch {
			// if we're tolerating mismatches, report the actual hash through different mechanisms.
			// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
			arena.hash = rio.CommitID(actualTreeHash)
		}
		// If tolerance mode not configured, this is a panic.
		panic(&def.ErrHashMismatch{
			Expected: def.Ware{Type: string(Kind), Hash: string(dataHash)},
			Actual:   def.Ware{Type: string(Kind), Hash: string(actualTreeHash)},
		})
	}, rio.TryPlanWhitelist)
	return arena
}

func (t S3Transmat) Scan(
	kind rio.TransmatKind,
	subjectPath string,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.CommitID {
	var commitID rio.CommitID
	meep.Try(func() {
		// Basic validation and config
		mixins.MustBeType(Kind, kind)
		config := rio.EvaluateConfig(options...)

		// If scan area doesn't exist, bail immediately.
		// No need to even start dialing warehouses if we've got nothing for em.
		_, err := os.Stat(subjectPath)
		if err != nil {
			if os.IsNotExist(err) {
				return // empty commitID
			} else {
				panic(err)
			}
		}

		// First... no save locations is a special case: still need to hash.
		if len(siloURIs) == 0 {
			// walk, fwrite, hash
			commitID = rio.CommitID(tartrans.Save(ioutil.Discard, subjectPath, config.FilterSet, hasherFactory))
			return // for no-save, that's it, we're done
		}

		// Load keys.
		keys := mustLoadKeys()
		// Dial warehouses.
		warehouses := make([]*Warehouse, 0, len(siloURIs))
		for _, uri := range siloURIs {
			wh := NewWarehouse(uri, keys)
			err := wh.PingWritable()
			if err == nil {
				warehouses = append(warehouses, wh)
			} else {
				log.Info("Unable to contact a warehouse, skipping it",
					"warehouse", uri,
					"reason", err,
				)
			}
		}
		// By default we're tolerant of some warehouses being unresponsive
		//  (mirroring is easy and conflict free, after all), but if
		//   ALL of them are down?  That's bad enough news to stop for.
		if len(warehouses) == 0 {
			panic(&def.ErrWarehouseUnavailable{
				Msg:    "No warehouses responded!",
				During: "save",
			})
		}

		// Open output streams for writing.
		// Since these are all behaving as just one `io.Writer` stream, this could maybe be factored out.
		// Error handling is currently "anything -> panic".  This should probably be more resilient.  (That might need another refactor so we have an upload call per remote.)
		controllers := make([]*writeController, 0)
		writers := make([]io.Writer, 0)
		for _, wh := range warehouses {
			controller := wh.openWriter()
			controllers = append(controllers, controller)
			writers = append(writers, controller.writer)
		}
		stream := io.MultiWriter(writers...)
		if len(writers) < 1 {
			stream = ioutil.Discard
		}

		// walk, fwrite, hash
		commitID = rio.CommitID(tartrans.Save(stream, subjectPath, config.FilterSet, hasherFactory))

		// commit
		for _, controller := range controllers {
			controller.Commit(commitID)
		}
	}, rio.TryPlanWhitelist)
	return commitID
}

func mustLoadKeys() s3gof3r.Keys {
	// TODO someday URIs should grow smart enough to control this in a more general fashion -- but for now, host ENV is actually pretty feasible and plays easily with others.
	// TODO read-only should not necessarily require keys! anon access *may* be 100% valid.
	//   Buuuuut s3gof3r doesn't seem to understand empty keys; it still sends them as if to login, and AWS says 403.  So, foo.
	keys, err := s3gof3r.EnvKeys()
	if err == nil {
		return keys
	}
	panic(&def.ErrConfigValidation{
		Msg: "s3 credentials missing.  set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.",
	})
}

type dirArena struct {
	path string
	hash rio.CommitID
}

func (a dirArena) Path() string {
	return a.path
}

func (a dirArena) Hash() rio.CommitID {
	return a.hash
}

// rm's.
// does not consider it an error if path already does not exist.
func (a dirArena) Teardown() {
	if err := os.RemoveAll(a.path); err != nil {
		if e2, ok := err.(*os.PathError); ok && e2.Err == syscall.ENOENT && e2.Path == a.path {
			return
		}
		panic(meep.Meep(
			&rio.ErrInternal{Msg: "Failed to tear down arena"},
			meep.Cause(err),
		))
	}
}
