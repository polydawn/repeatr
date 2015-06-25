package tar

import (
	"archive/tar"
	"encoding/base64"
	"hash"
	"io"
	"path"
	"strings"
	"time"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/filter"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
	"polydawn.net/repeatr/lib/treewalk"
)

// Walks `basePath`, hashing it, pushing the encoded tar to `file`, and returning the final hash.
func Save(file io.Writer, basePath string, filterset filter.FilterSet, hasherFactory func() hash.Hash) string {
	// walk filesystem, copying and accumulating data for integrity check
	bucket := &fshash.MemoryBucket{}
	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()
	if err := saveWalk(basePath, tarWriter, filterset, bucket, hasherFactory); err != nil {
		panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
	}

	// hash whole tree
	actualTreeHash := fshash.Hash(bucket, hasherFactory)

	// report
	return base64.URLEncoding.EncodeToString(actualTreeHash)
}

func saveWalk(srcBasePath string, tw *tar.Writer, filterset filter.FilterSet, bucket fshash.Bucket, hasherFactory func() hash.Hash) error {
	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}
		hdr, file := fs.ScanFile(srcBasePath, filenode.Path, filenode.Info)
		// apply filters.  on scans, this is pretty easy, all of em just apply to the stream in memory.
		hdr = filterset.Apply(hdr)
		// flaten time to seconds.  this tar writer impl doesn't do subsecond precision.
		// the writer will flatten it internally of course, but we need to do it here as well
		// so that the hash and the serial form are describing the same thing.
		hdr.ModTime = hdr.ModTime.Truncate(time.Second)
		wat := tar.Header(hdr) // this line is... we're not gonna talk about this.
		tw.WriteHeader(&wat)
		if file == nil {
			bucket.Record(hdr, nil)
		} else {
			defer file.Close()
			hasher := hasherFactory()
			tee := io.MultiWriter(tw, hasher)
			_, err := io.Copy(tee, file)
			if err != nil {
				return err
			}
			bucket.Record(hdr, hasher.Sum(nil))
		}
		return nil
	}
	return fs.Walk(srcBasePath, preVisit, nil)
}

func Extract(tr *tar.Reader, destBasePath string, bucket fshash.Bucket, hasherFactory func() hash.Hash) {
	for {
		thdr, err := tr.Next()
		if err == io.EOF {
			break // end of archive
		}
		if err != nil {
			panic(integrity.WarehouseConnectionError.New("corrupt tar: %s", err))
		}
		hdr := fs.Metadata(*thdr)
		// filter/sanify values:
		// - names must be clean, relative dot-slash prefixed, and dirs slash-suffixed
		// - times should never be go's zero value; replace those with epoch
		// Note that names at this point should be handled by `path` (not `filepath`; these are canonical form for feed to hashing)
		hdr.Name = path.Clean(hdr.Name)
		if strings.HasPrefix(hdr.Name, "../") {
			panic(integrity.WarehouseConnectionError.New("corrupt tar: paths that use '../' to leave the base dir are invalid"))
		}
		if hdr.Name != "." {
			hdr.Name = "./" + hdr.Name
		}
		if hdr.ModTime.IsZero() {
			hdr.ModTime = def.Epochwhen
		}
		if hdr.AccessTime.IsZero() {
			hdr.AccessTime = def.Epochwhen
		}
		// place the file
		switch hdr.Typeflag {
		case tar.TypeReg:
			reader := &flak.HashingReader{tr, hasherFactory()}
			fs.PlaceFile(destBasePath, hdr, reader)
			bucket.Record(hdr, reader.Hasher.Sum(nil))
		case tar.TypeDir:
			hdr.Name += "/"
			fallthrough
		default:
			fs.PlaceFile(destBasePath, hdr, nil)
			bucket.Record(hdr, nil)
		}
	}
	// cleanup dir times with a post-order traversal over the bucket
	if err := treewalk.Walk(bucket.Iterator(), nil, func(node treewalk.Node) error {
		record := node.(fshash.RecordIterator).Record()
		if record.Metadata.Typeflag == tar.TypeDir {
			fs.PlaceDirTime(destBasePath, record.Metadata)
		}
		return nil
	}); err != nil {
		panic(err)
	}
}
