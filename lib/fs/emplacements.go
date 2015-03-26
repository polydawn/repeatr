package fs

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/lib/fspatch"
)

func PlaceFile(destBasePath string, hdr Metadata, body io.Reader) {
	// 'destBasePath' should be an absolute path on the host.
	// 'hdr.Name' should be the full relative path of the file.
	// if it has an absolute prefix, that's quietly ignored, and it's treated as relative anyway.

	destPath := filepath.Join(destBasePath, hdr.Name)
	mode := hdr.FileMode()

	switch hdr.Typeflag {
	case tar.TypeDir:
		if hdr.Name == "./" {
			// for the base dir only:
			// the dir may exist; we'll just chown+chmod+chtime it.
			// there is no race-free path through this btw, unless you know of a way to lstat and mkdir in the same syscall.
			if fi, err := os.Lstat(destPath); err == nil && fi.IsDir() {
				break
			}
		}
		if err := os.Mkdir(destPath, mode); err != nil {
			ioError(err)
		}
	case tar.TypeReg, tar.TypeRegA:
		file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, mode)
		if err != nil {
			ioError(err)
		}
		if _, err := io.Copy(file, body); err != nil {
			file.Close()
			ioError(err)
		}
		file.Close()
	case tar.TypeSymlink:
		targetPath := filepath.Join(filepath.Dir(destPath), hdr.Linkname)
		if !strings.HasPrefix(targetPath, destBasePath) {
			panic(BreakoutError.New("invalid symlink %q -> %q", targetPath, hdr.Linkname))
		}
		if err := os.Symlink(hdr.Linkname, destPath); err != nil {
			ioError(err)
		}
	case tar.TypeLink:
		targetPath := filepath.Join(destBasePath, hdr.Linkname)
		if !strings.HasPrefix(targetPath, destBasePath) {
			panic(BreakoutError.New("invalid hardlink %q -> %q", targetPath, hdr.Linkname))
		}
		if err := os.Link(targetPath, destPath); err != nil {
			ioError(err)
		}
	case tar.TypeBlock:
		mode := uint32(hdr.Mode&07777) | syscall.S_IFBLK
		if err := syscall.Mknod(destPath, mode, int(fspatch.Mkdev(hdr.Devmajor, hdr.Devminor))); err != nil {
			ioError(err)
		}
	case tar.TypeChar:
		mode := uint32(hdr.Mode&07777) | syscall.S_IFCHR
		if err := syscall.Mknod(destPath, mode, int(fspatch.Mkdev(hdr.Devmajor, hdr.Devminor))); err != nil {
			ioError(err)
		}
	case tar.TypeFifo:
		if err := syscall.Mkfifo(destPath, uint32(hdr.Mode&07777)); err != nil {
			ioError(err)
		}
	default:
		panic(errors.NotImplementedError.New("The tennants of filesystems have changed!  We're not prepared for this file mode %q", hdr.Typeflag))
	}

	if err := os.Lchown(destPath, hdr.Uid, hdr.Gid); err != nil {
		ioError(err)
	}

	for key, value := range hdr.Xattrs {
		if err := fspatch.Lsetxattr(destPath, key, []byte(value), 0); err != nil {
			ioError(err)
		}
	}

	if hdr.Typeflag != tar.TypeSymlink {
		// do this for everything not a symlink, since there's no such thing as `lchmod` on linux -.-
		if err := os.Chmod(destPath, mode); err != nil {
			ioError(err)
		}
	}

	if err := fspatch.LUtimesNano(destPath, hdr.AccessTime, hdr.ModTime); err != nil {
		ioError(err)
	}
}
