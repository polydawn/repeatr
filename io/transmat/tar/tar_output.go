package tar

import (
	"archive/tar"
	"encoding/base64"
	"hash"
	"io"
	"time"

	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fshash"
)

// Walks `basePath`, hashing it, pushing the encoded tar to `file`, and returning the final hash.
func Save(file io.Writer, basePath string, hasherFactory func() hash.Hash) string {
	// walk filesystem, copying and accumulating data for integrity check
	bucket := &fshash.MemoryBucket{}
	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()
	if err := saveWalk(basePath, tarWriter, bucket, hasherFactory); err != nil {
		panic(err) // TODO this is not well typed, and does not clearly indicate whether scanning or committing had the problem
	}

	// hash whole tree
	actualTreeHash, _ := fshash.Hash(bucket, hasherFactory)

	// report
	return base64.URLEncoding.EncodeToString(actualTreeHash)
}

func saveWalk(srcBasePath string, tw *tar.Writer, bucket fshash.Bucket, hasherFactory func() hash.Hash) error {
	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}
		hdr, file := fs.ScanFile(srcBasePath, filenode.Path, filenode.Info)
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
