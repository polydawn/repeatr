package fshash

import (
	"archive/tar"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/fspatch"
)

func FillBucket(srcBasePath, destBasePath string, bucket Bucket, hasherFactory func() hash.Hash) error {
	// If copying: Dragons: you can set atime and you can set mtime, but you can't ever set ctime again.
	// Filesystem APIs are constructed such that it's literally impossible to do an attribute-preserving copy in userland.
	return filepath.Walk(srcBasePath, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := "." + strings.TrimPrefix(srcPath, srcBasePath)
		destPath := filepath.Join(destBasePath, relPath)
		mode := info.Mode()
		switch {
		case mode&os.ModeDir == os.ModeDir:
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = relPath
			hdr.ChangeTime = def.Somewhen
			hdr.AccessTime = def.Somewhen
			if destBasePath != "" {
				if err := os.Mkdir(destPath, mode&os.ModePerm); err != nil {
					return err
				}
				// FIXME: this needs post-order traversal to take useful effect
				if err := fspatch.UtimesNano(destPath, []syscall.Timespec{def.SomewhenTimespec, syscall.NsecToTimespec(hdr.ModTime.UnixNano())}); err != nil {
					return err
				}
			}
			bucket.Record(Metadata(*hdr), nil)
		case mode&os.ModeSymlink == os.ModeSymlink:
			var link string
			var err error
			if link, err = os.Readlink(srcPath); err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}
			hdr.Name = relPath
			hdr.ChangeTime = def.Somewhen
			hdr.AccessTime = def.Somewhen
			if destBasePath != "" {
				if err := os.Symlink(destPath, link); err != nil {
					return err
				}
				if err := fspatch.LUtimesNano(destPath, []syscall.Timespec{def.SomewhenTimespec, syscall.NsecToTimespec(hdr.ModTime.UnixNano())}); err != nil {
					return err
				}
			}
			bucket.Record(Metadata(*hdr), nil)
		case mode&os.ModeNamedPipe == os.ModeNamedPipe:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeSocket == os.ModeSocket:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeDevice == os.ModeDevice:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeCharDevice == os.ModeCharDevice:
			panic(errors.NotImplementedError.New("TODO"))
		case mode&os.ModeType == 0: // i.e. regular file
			// copy data into place and accumulate hash
			src, err := os.OpenFile(srcPath, os.O_RDONLY, 0)
			if err != nil {
				return err
			}
			defer src.Close()
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
			_, err = io.Copy(tee, src)
			if err != nil {
				return err
			}
			// marshal headers and save to bucket with hash
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = relPath
			hdr.ChangeTime = def.Somewhen
			hdr.AccessTime = def.Somewhen
			if destBasePath != "" {
				if err := fspatch.UtimesNano(destPath, []syscall.Timespec{def.SomewhenTimespec, syscall.NsecToTimespec(hdr.ModTime.UnixNano())}); err != nil {
					return err
				}
			}
			hash := hasher.Sum(nil)
			bucket.Record(Metadata(*hdr), hash)
		default:
			panic(errors.NotImplementedError.New("The tennants of filesystems have changed!  We're not prepared for this file mode %d", mode))
			// side note: i don't know how to check for hardlinks
			// except for by `os.SameFile` but that obviously doesn't scale.
			// so, none of our hashing definitions can accept hardlinks :/
			// we could add a hash of inodes to bucket to address this.
		}
		return nil
	})
}
