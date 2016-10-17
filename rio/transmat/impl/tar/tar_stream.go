package tar

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/flak"
	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/lib/fshash"
	"go.polydawn.net/repeatr/lib/treewalk"
	"go.polydawn.net/repeatr/rio/filter"
)

/*
	Walks `basePath`, hashing it, encoding the contents as a tar and sending the gzip'd
	stream to `file`, and returning the final hash after all files have been walked.
*/
func Save(file io.Writer, basePath string, filterset filter.FilterSet, hasherFactory func() hash.Hash) string {
	// Stream the tar and compress on the way out.
	//  Note on compression levels: The default is 6; and per per http://tukaani.org/lzma/benchmarks.html
	//  this appears quite reasonable: higher levels appear to have minimal size payoffs, but significantly rising compress time costs;
	//  decompression time does not vary with compression level.
	// Save a gzip reference just to close it; tar.Writer doesn't passthru its own close.
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()
	// walk filesystem, copying and accumulating data for integrity check
	bucket := &fshash.MemoryBucket{}
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

func Extract(tr *tar.Reader, destBasePath string, bucket fshash.Bucket, hasherFactory func() hash.Hash, log log15.Logger) {
	for {
		thdr, err := tr.Next()
		if err == io.EOF {
			break // end of archive
		}
		if err != nil {
			panic(&def.ErrWareCorrupt{
				Msg: fmt.Sprintf("corrupt tar: %s", err),
				// Ware // TODO we should be able to get, this abstraction is just silly
				// From // TODO plz
			})
		}
		hdr := fs.Metadata(*thdr)
		// filter/sanify values:
		// - names must be clean, relative dot-slash prefixed, and dirs slash-suffixed
		// - times should never be go's zero value; replace those with epoch
		// Note that names at this point should be handled by `path` (not `filepath`; these are canonical form for feed to hashing)
		hdr.Name = path.Clean(hdr.Name)
		if strings.HasPrefix(hdr.Name, "../") {
			panic(&def.ErrWareCorrupt{
				Msg: "corrupt tar: paths that use '../' to leave the base dir are invalid",
				// Ware // TODO we should be able to get, this abstraction is just silly
				// From // TODO plz
			})
		}
		if hdr.Name != "." {
			hdr.Name = "./" + hdr.Name
		}
		if hdr.ModTime.IsZero() {
			hdr.ModTime = fs.Epochwhen
		}
		if hdr.AccessTime.IsZero() {
			hdr.AccessTime = fs.Epochwhen
		}
		// conjure parents, if necessary.  tar format allows implicit parent dirs.
		// Note that if any of the implicitly conjured dirs is specified later, unpacking won't notice,
		// but bucket hashing iteration will (correctly) blow up for repeat entries.
		// It may well be possible to construct a tar like that, but it's already well established that
		// tars with repeated filenames are just asking for trouble and shall be rejected without
		// ceremony because they're just a ridiculous idea.
		parts := strings.Split(hdr.Name, "/")
		for i := range parts[:len(parts)-1] {
			i++
			_, err := os.Lstat(filepath.Join(append([]string{destBasePath}, parts[:i]...)...))
			// if it already exists, move along; if the error is anything interesting, let PlaceFile decide how to deal with it
			if err == nil || !os.IsNotExist(err) {
				continue
			}
			// if we're missing a dir, conjure a node with defaulted values (same as we do for "./")
			conjuredHdr := fshash.DefaultDirRecord().Metadata
			conjuredHdr.Name = strings.Join(parts[:i], "/") + "/" // path.Join does cleaning; unwanted.
			fs.PlaceFile(destBasePath, conjuredHdr, nil)
			bucket.Record(conjuredHdr, nil)
		}
		// place the file
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			reader := &flak.HashingReader{tr, hasherFactory()}
			hdr.Typeflag = tar.TypeReg
			fs.PlaceFile(destBasePath, hdr, reader)
			bucket.Record(hdr, reader.Hasher.Sum(nil))
		case tar.TypeDir:
			hdr.Name += "/"
			fs.PlaceFile(destBasePath, hdr, nil)
			bucket.Record(hdr, nil)
		case tar.TypeSymlink, tar.TypeLink, tar.TypeBlock, tar.TypeChar, tar.TypeFifo:
			fs.PlaceFile(destBasePath, hdr, nil)
			bucket.Record(hdr, nil)
		case tar.TypeCont, tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName, tar.TypeGNULongLink, tar.TypeGNUSparse:
			log.Warn(fmt.Sprintf("tar extract: ignoring entry type %q", hdr.Typeflag))
		default:
			panic(errors.NotImplementedError.New("Unknown file mode %q", hdr.Typeflag))
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
