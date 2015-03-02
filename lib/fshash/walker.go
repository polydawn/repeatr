package fshash

import (
	"archive/tar"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spacemonkeygo/errors"
)

func FillBucket(srcPath, destPath string, bucket Bucket, hasherFactory func() hash.Hash) error {
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		path = "." + strings.TrimPrefix(path, srcPath)
		if err != nil {
			return err
		}
		mode := info.Mode()
		// TODO special handling for root dir? ... it's just path="" right now
		switch {
		case mode&os.ModeDir == os.ModeDir:
			if err := os.Mkdir(filepath.Join(destPath, path), mode&os.ModePerm); err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			// TODO: normalize name
			bucket.Record(Metadata(*hdr), nil)
		case mode&os.ModeSymlink == os.ModeSymlink:
			var link string
			var err error
			if link, err = os.Readlink(path); err != nil {
				return err
			}
			if err := os.Symlink(filepath.Join(destPath, path), link); err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}
			// TODO: normalize name
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
			src, err := os.OpenFile(filepath.Join(srcPath, path), os.O_RDONLY, 0)
			if err != nil {
				return err
			}
			defer src.Close()
			dest, err := os.OpenFile(filepath.Join(destPath, path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode&os.ModePerm)
			if err != nil {
				return err
			}
			defer dest.Close()
			hasher := hasherFactory()
			tee := io.MultiWriter(dest, hasher)
			_, err = io.Copy(tee, src)
			if err != nil {
				return err
			}
			// marshal headers and save to bucket with hash
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			// TODO: normalize name
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
