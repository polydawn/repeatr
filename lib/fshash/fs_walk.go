package fshash

import (
	"hash"
	"io"
	"path/filepath"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io/filter"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fspatch"
)

func FillBucket(srcBasePath, destBasePath string, bucket Bucket, filterset filter.FilterSet, hasherFactory func() hash.Hash) error {
	// If copying: Dragons: you can set atime and you can set mtime, but you can't ever set ctime again.
	// Filesystem APIs are constructed such that it's literally impossible to do an attribute-preserving copy in userland.

	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}

		// scan source attributes
		hdr, file := fs.ScanFile(srcBasePath, filenode.Path, filenode.Info)

		// apply filters.  on scans, this is pretty easy, all of em just apply to the stream in memory.
		// NOT YET SUPPORTED for use in materialize.  (well, it'll work actually, just... not optimally.)
		hdr = filterset.Apply(hdr)

		// write headers to the hash bucket, and if applicable copy file content to new path
		if file == nil {
			if destBasePath != "" {
				fs.PlaceFile(destBasePath, hdr, nil)
			}
			bucket.Record(hdr, nil)
		} else {
			defer file.Close()
			hasher := hasherFactory()
			if destBasePath != "" {
				reader := &flak.HashingReader{file, hasher}
				fs.PlaceFile(destBasePath, hdr, reader)
			} else {
				io.Copy(hasher, file)
			}
			bucket.Record(hdr, hasher.Sum(nil))
		}
		return nil
	}
	postVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Info.IsDir() && destBasePath != "" {
			// XXX: this is looking back on the fileinfo instead of the header and punting on atime with a hack.
			// this would be better if fs.FilewalkNode supported an attachment so we could stick the header on, but in practice, same values.
			if err := fspatch.UtimesNano(filepath.Join(destBasePath, filenode.Path), def.Epochwhen, filenode.Info.ModTime()); err != nil {
				return err
			}
		}
		return nil
	}

	return fs.Walk(srcBasePath, preVisit, postVisit)
}
