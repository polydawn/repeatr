package fshash

import (
	"hash"
	"io"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
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
		switch {
		case mode&os.ModeDir == os.ModeDir:
			fallthrough
		case mode&os.ModeSymlink == os.ModeSymlink:
			if destBasePath != "" {
				fs.PlaceFile(destBasePath, hdr, nil)
			}
			bucket.Record(hdr, nil)
		case mode&os.ModeNamedPipe == os.ModeNamedPipe:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeSocket == os.ModeSocket:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeDevice == os.ModeDevice:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeCharDevice == os.ModeCharDevice:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeType == 0: // i.e. regular file
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
		default:
			panic(errors.NotImplementedError.New("The tennants of filesystems have changed!  We're not prepared for this file mode %d", mode))
			// side note: i don't know how to check for hardlinks
			// except for by `os.SameFile` but that obviously doesn't scale.
			// so, none of our hashing definitions can accept hardlinks :/
			// we could add a hash of inodes to bucket to address this.
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
