package fshash

import (
	"hash"
	"io"
	"os"
	"path/filepath"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/fs"
	"polydawn.net/repeatr/lib/fspatch"
)

func FillBucket(srcBasePath, destBasePath string, bucket Bucket, hasherFactory func() hash.Hash) error {
	// If copying: Dragons: you can set atime and you can set mtime, but you can't ever set ctime again.
	// Filesystem APIs are constructed such that it's literally impossible to do an attribute-preserving copy in userland.

	preVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Err != nil {
			return filenode.Err
		}
		destPath := filepath.Join(destBasePath, filenode.Path)
		mode := filenode.Info.Mode()
		hdr, file := fs.ScanFile(srcBasePath, filenode.Path, filenode.Info)
		if file == nil {
			if destBasePath != "" {
				fs.PlaceFile(destBasePath, hdr, nil)
			}
			bucket.Record(hdr, nil)
		} else {
			// TODO : rearrange hasher stream so we can call lib/fs.PlaceFile
			// copy data into place and accumulate hash
			defer file.Close()
			hasher := hasherFactory()
			var tee io.Writer
			if destBasePath != "" {
				dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode&os.ModePerm)
				if err != nil {
					return err
				}
				defer dest.Close()
				tee = io.MultiWriter(dest, hasher)
			} else {
				tee = hasher
			}
			_, err := io.Copy(tee, file)
			if err != nil {
				return err
			}
			// marshal headers and save to bucket with hash
			if destBasePath != "" {
				if err := fspatch.UtimesNano(destPath, def.Somewhen, hdr.ModTime); err != nil {
					return err
				}
				if err := os.Chown(destPath, hdr.Uid, hdr.Gid); err != nil {
					return err
				}
			}
			hash := hasher.Sum(nil)
			bucket.Record(hdr, hash)
		}
		return nil
	}
	postVisit := func(filenode *fs.FilewalkNode) error {
		if filenode.Info.IsDir() && destBasePath != "" {
			if err := fspatch.UtimesNano(filepath.Join(destBasePath, filenode.Path), def.Somewhen, filenode.Info.ModTime()); err != nil {
				return err
			}
		}
		return nil
	}

	return fs.Walk(srcBasePath, preVisit, postVisit)
}
