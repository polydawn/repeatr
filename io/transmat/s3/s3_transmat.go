package s3

import (
	"archive/tar"
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/rlmcpherson/s3gof3r"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/io"
	tartrans "polydawn.net/repeatr/io/transmat/tar"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/lib/guid"
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
		panic(rio.TransmatError.New("Unable to set up workspace: %s", err))
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
			case "s3":
				warehouseCtntAddr = false
			case "s3+splay": // deprecated
				warehouseCtntAddr = true
			case "s3+ca":
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

		// load keys from env
		// TODO someday URIs should grow smart enough to control this in a more general fashion -- but for now, host ENV is actually pretty feasible and plays easily with others.
		// TODO should not require keys!  we're just reading, after all; anon access is 100% valid.
		//   Buuuuut s3gof3r doesn't seem to understand empty keys; it still sends them as if to login, and AWS says 403.  So, foo.
		keys, err := s3gof3r.EnvKeys()
		if err != nil {
			panic(S3CredentialsMissingError.Wrap(err))
		}

		// initialize reader from s3!
		getPath := warehousePathPrefix
		if warehouseCtntAddr {
			getPath = path.Join(warehousePathPrefix, string(dataHash))
		}
		s3reader := makeS3reader(warehouseBucketName, getPath, keys)
		defer s3reader.Close()
		// prepare decompression as necessary
		reader, err := tartrans.Decompress(s3reader)
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
		actualTreeHash := fshash.Hash(bucket, hasherFactory)

		// verify total integrity
		expectedTreeHash, err := base64.URLEncoding.DecodeString(string(dataHash))
		if err != nil {
			panic(rio.ConfigError.New("Could not parse hash: %s", err))
		}
		if bytes.Equal(actualTreeHash, expectedTreeHash) {
			// excellent, got what we asked for.
			arena.hash = dataHash
		} else {
			// this may or may not be grounds for panic, depending on configuration.
			if config.AcceptHashMismatch {
				// if we're tolerating mismatches, report the actual hash through different mechanisms.
				// you probably only ever want to use this in tests or debugging; in prod it's just asking for insanity.
				arena.hash = rio.CommitID(actualTreeHash)
			} else {
				panic(rio.NewHashMismatchError(string(dataHash), base64.URLEncoding.EncodeToString(actualTreeHash)))
			}
		}
	}).Catch(rio.Error, func(err *errors.Error) {
		panic(err)
	}).CatchAll(func(err error) {
		panic(rio.UnknownError.Wrap(err))
	}).Done()
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

		// load keys from env
		// TODO someday URIs should grow smart enough to control this in a more general fashion -- but for now, host ENV is actually pretty feasible and plays easily with others.
		keys, err := s3gof3r.EnvKeys()
		if err != nil {
			panic(S3CredentialsMissingError.Wrap(err))
		}

		// Parse URI; Find warehouses; Open output streams for writing.
		// Since these are all behaving as just one `io.Writer` stream, this could maybe be factored out.
		// Error handling is currently "anything -> panic".  This should probably be more resilient.  (That might need another refactor so we have an upload call per remote.)
		// TODO : both this and the tar code that has a similar single stream idea should use an interface
		//  And that interface should have a concept of mv so we can make atomic commits.
		//  I'm not doing multiple URIs here until we get that, because the io.Writer interface just
		//   doesn't cut it like it did for tars (and really, it's ignoring a major issue to use it there, too).
		//  ...Fuck it, we're gonna do it
		controllers := make([]*s3warehousePut, 0)
		writers := make([]io.Writer, 0) // this is dumb, but we end up making one of these to satisfy the type conversation for MultiWriter anyway
		for _, givenURI := range siloURIs {
			u, err := url.Parse(string(givenURI))
			if err != nil {
				panic(rio.ConfigError.New("failed to parse URI: %s", err))
			}
			controller := &s3warehousePut{}
			controller.bucketName = u.Host
			controller.pathPrefix = u.Path
			var ctntAddr bool
			switch u.Scheme {
			case "s3":
				ctntAddr = false
			case "s3+splay": // deprecated
				ctntAddr = true
			case "s3+ca":
				ctntAddr = true
			default:
				panic(rio.ConfigError.New("unrecognized scheme: %q", u.Scheme))
			}
			// dial it and initialize writer to s3!
			// if the URI indicated CA behavior, first stream data to {$bucketName}:{dirname($storePath)}/.tmp.upload.{basename($storePath)}.{random()};
			// this allows us to start uploading before the final hash is determined and relocate it later.
			// for direct paths, upload into place, because aws already manages atomicity at that scale (and they don't have a rename or copy operation that's free, because uh...?  no time to implement it since 2006, apparently).
			controller.keys = keys
			if ctntAddr {
				controller.tmpPath = path.Join(
					path.Dir(controller.pathPrefix),
					".tmp.upload."+path.Base(controller.pathPrefix)+"."+guid.New(),
				)
				controller.stream = makeS3writer(controller.bucketName, controller.tmpPath, keys)
			} else {
				controller.stream = makeS3writer(controller.bucketName, controller.pathPrefix, keys)
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

type s3warehousePut struct {
	stream     io.WriteCloser
	keys       s3gof3r.Keys
	bucketName string
	pathPrefix string
	tmpPath    string // if set, using content-addressible mode.
}

/*
	Fsync's the stream, and does the commit mv into place
	using `hash` if in content-addressable mode.
*/
func (wp *s3warehousePut) Commit(hash string) {
	// flush and check errors on the final write to s3.
	// be advised that this close method does *a lot* of work aside from connection termination.
	// also calling it twice causes the library to wigg out and delete things, i don't even.
	if err := wp.stream.Close(); err != nil {
		panic(rio.WarehouseIOError.Wrap(err))
	}

	// if the URI indicated CA behavior, rename the temp filepath to the real one;
	// the upload location is suffixed to make a CA resting place.
	if wp.tmpPath != "" {
		finalPath := path.Join(wp.pathPrefix, hash)
		reloc(wp.bucketName, wp.tmpPath, finalPath, wp.keys)
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
