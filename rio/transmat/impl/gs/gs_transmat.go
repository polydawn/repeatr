package gs

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"golang.org/x/oauth2"

	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/lib/fshash"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rio"
	tartrans "go.polydawn.net/repeatr/rio/transmat/impl/tar"
)

const Kind = rio.TransmatKind("gs")

var _ rio.Transmat = &GsTransmat{}

type GsTransmat struct {
	workPath string
}

var _ rio.TransmatFactory = New

func New(workPath string) rio.Transmat {
	err := os.MkdirAll(workPath, 0755)
	if err != nil {
		panic(rio.TransmatError.New("Unable to set up workspace: %s", err))
	}
	return &GsTransmat{workPath}
}

var hasherFactory = sha512.New384

/*
	URL Scheme for Google cloud storage buckets.
	e.g. gs://bucket-name/object-name
*/
const SCHEME_GS = "gs"

/*
	URL Scheme for Google cloud storage buckets using content addressable objects
	e.g. gs+ca://bucket-name/object-name-hash
*/
const SCHEME_GS_CAS = "gs+ca"

/*
	Arenas produced by Dir Transmats may be relocated by simple `mv`.
*/
func (t *GsTransmat) Materialize(
	kind rio.TransmatKind,
	dataHash rio.CommitID,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.Arena {
	var arena dirArena
	try.Do(func() {
		// Basic validation and config
		config := rio.EvaluateConfig(options...)
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

		// Parse URI; Find warehouses.
		if len(siloURIs) < 1 {
			panic(rio.ConfigError.New("Materialization requires at least one data source!"))
			// Note that it's possible a caching layer will satisfy things even without data sources...
			//  but if that was going to happen, it already would have by now.
		}
		// Our policy is to take the first path that exists.
		//  This lets you specify a series of potential locations, and if one is unavailable we'll just take the next.
		var warehouseBucketName string
		var warehousePathPrefix string
		var warehouseCtntAddr bool
		for _, givenURI := range siloURIs {
			u, err := url.Parse(string(givenURI))
			if err != nil {
				panic(rio.ConfigError.New("failed to parse URI: %s", err))
			}
			warehouseBucketName = u.Host
			warehousePathPrefix = u.Path
			switch u.Scheme {
			case SCHEME_GS:
				warehouseCtntAddr = false
			case SCHEME_GS_CAS:
				warehouseCtntAddr = true
			default:
				panic(rio.ConfigError.New("unrecognized scheme: %q", u.Scheme))
			}
			// TODO figure out how to check for data (or at least warehouse!) presence;
			//  currently just assuming the first one's golden, and blowing up later if it's not.
			break
		}
		if warehouseBucketName == "" {
			panic(rio.WarehouseUnavailableError.New("No warehouses were available!"))
		}

		token, err := GetAccessToken()
		if err != nil {
			panic(GsCredentialsMissingError.Wrap(err))
		}

		// initialize reader
		getPath := warehousePathPrefix
		if warehouseCtntAddr {
			getPath = path.Join(warehousePathPrefix, string(dataHash))
		}
		gsReader := makeGsReader(warehouseBucketName, getPath, token)
		defer gsReader.Close()

		// prepare decompression as necessary
		reader, err := tartrans.Decompress(gsReader)
		if err != nil {
			panic(rio.WarehouseCorruptionError.New("could not start decompressing: %s", err))
		}
		tarReader := tar.NewReader(reader)

		// Create staging arena to produce data into.
		arena.path, err = ioutil.TempDir(t.workPath, "")
		if err != nil {
			panic(rio.TransmatError.New("Unable to create arena: %s", err))
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
		if actualTreeHash == expectedTreeHash {
			// excellent, got what we asked for.
			arena.hash = dataHash
		} else {
			// this may or may not be grounds for panic, depending on configuration.
			if config.AcceptHashMismatch {
				// if we're tolerating mismatches, report the actual hash through different mechanisms.
				// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
				arena.hash = rio.CommitID(actualTreeHash)
			} else {
				panic(rio.NewHashMismatchError(string(dataHash), actualTreeHash))
			}
		}
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return arena
}

func (t GsTransmat) Scan(
	kind rio.TransmatKind,
	subjectPath string,
	siloURIs []rio.SiloURI,
	log log15.Logger,
	options ...rio.MaterializerConfigurer,
) rio.CommitID {
	var commitID rio.CommitID
	try.Do(func() {
		// Basic validation and config
		config := rio.EvaluateConfig(options...)
		if kind != Kind {
			panic(errors.ProgrammerError.New("This transmat supports definitions of type %q, not %q", Kind, kind))
		}

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

		token, err := GetAccessToken()
		if err != nil {
			panic(GsCredentialsMissingError.Wrap(err))
		}

		// I honestly don't know what most of this does. It was part of the AWS code and so I left it.
		// That said it appears to be related to composite object support, which is available in GCS:
		// ```
		//    To support parallel uploads and limited append/edit functionality,
		//    Google Cloud Storage allows users to compose up to 32 existing objects
		//    into a new object without transferring additional object data.
		// ```
		// Look here for more information: https://cloud.google.com/storage/docs/composite-objects
		// TODO: Composite object support with GCS
		numSilos := len(siloURIs)
		controllers := make([]*gsWarehousePut, 0, numSilos)
		writers := make([]io.Writer, 0, numSilos)
		for _, givenURI := range siloURIs {
			u, err := url.Parse(string(givenURI))
			if err != nil {
				panic(rio.ConfigError.New("failed to parse URI: %s", err))
			}
			controller := &gsWarehousePut{}
			controller.bucketName = u.Host
			controller.pathPrefix = u.Path
			var ctntAddr bool
			switch u.Scheme {
			case SCHEME_GS:
				ctntAddr = false
			case SCHEME_GS_CAS:
				ctntAddr = true
			default:
				panic(rio.ConfigError.New("unrecognized scheme: %q", u.Scheme))
			}
			// if the URI indicated CA behavior, first stream data to {$bucketName}:{dirname($storePath)}/.tmp.upload.{basename($storePath)}.{random()};
			// this allows us to start uploading before the final hash is determined and relocate it later.
			controller.token = token
			if ctntAddr {
				controller.tmpPath = path.Join(
					path.Dir(controller.pathPrefix),
					".tmp.upload."+path.Base(controller.pathPrefix)+"."+guid.New(),
				)
				controller.stream, controller.errors = makeGsWriter(controller.bucketName, controller.tmpPath, token)
			} else {
				controller.stream, controller.errors = makeGsWriter(controller.bucketName, controller.pathPrefix, token)
			}
			controllers = append(controllers, controller)
			writers = append(writers, controller.stream)
		}
		stream := io.MultiWriter(writers...)
		if len(writers) < 1 {
			stream = ioutil.Discard
		}

		// walk, fwrite, hash
		commitID = rio.CommitID(tartrans.Save(stream, subjectPath, config.FilterSet, hasherFactory))

		// commit
		for _, controller := range controllers {
			controller.Commit(string(commitID))
		}
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
	return commitID
}

type gsWarehousePut struct {
	stream     io.WriteCloser
	bucketName string
	pathPrefix string
	tmpPath    string // if set, using content-addressible mode.
	token      *oauth2.Token
	errors     <-chan error
}

/*
	Fsync's the stream, and does the commit mv into place
	using `hash` if in content-addressable mode.
*/
func (wp *gsWarehousePut) Commit(hash string) {
	// flush and check errors on the final write
	// be advised that this close method does *a lot* of work aside from connection termination.
	// also calling it twice causes the library to wigg out and delete things, i don't even.
	if err := wp.stream.Close(); err != nil {
		panic(rio.WarehouseIOError.Wrap(err))
	}
	for err := range wp.errors {
		panic(rio.UnknownError.Wrap(err))
	}
	//TODO: We could check Cloud storage's content hash against ours to check for transport errors.

	// if the URI indicated CA behavior, rename the temp filepath to the real one;
	// the upload location is suffixed to make a CA resting place.
	if wp.tmpPath != "" {
		finalPath := path.Join(wp.pathPrefix, hash)
		reloc(wp.bucketName, wp.tmpPath, finalPath, wp.token)
	}
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
		panic(rio.TransmatError.New("Failed to tear down arena: %s", err))
	}
}
