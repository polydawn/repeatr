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

/*
	Places a file on the filesystem.
	Replicates all attributes described in the metadata.

	The path is the join of `destBasePath` and `hdr.Name`.
	`hdr.Name` should be the full relative path of the file;
	if it has an absolute prefix, that's quietly ignored, and it's treated as relative anyway.
	`hdr.Name` should match `filepath.Clean` output, except that it must always
	use the unix directory separator.

	No changes are allowed to occur outside of `destBasePath`.
	Hardlinks may not point outside of the base path.
	Symlinks may *point* at paths outside of the base path (because you
	may be about to chroot into this, in which case absolute link paths
	make perfect sense), and invalid symlinks are acceptable -- however
	symlinks may *not* be traversed during any part of `hdr.Name`; this is
	considered malformed input and will result in a BreakoutError.

	`destBasePath` MUST be absolute.  Isolation checks assume this, and
	have undefined operation if this requirement is not met.

	Please note that like all filesystem operations within a lightyear of
	symlinks, all validations are best-effort, but are only capable of
	correctness in the absense of concurrent modifications inside `destBasePath`.

	Device files *will* be created, with their maj/min numbers.
	This may be considered a security concern; you should whitelist inputs
	if using this to provision a sandbox.
*/
func PlaceFile(destBasePath string, hdr Metadata, body io.Reader) {
	destPath := filepath.Join(destBasePath, hdr.Name)
	mode := hdr.FileMode()

	// First, no part of the path may be a symlink.
	// We *could* create an application-level jailing effect as we walk this,
	// but that's just complicated enough to be dangerous, and also still
	// results in a world where results would vary depending on order of `PlaceFile` calls.
	// So!  Traversing symlinks during placement is fiat unacceptable.
	parts := strings.Split(hdr.Name, "/")
	for i := range parts {
		target, err := os.Readlink(filepath.Join(append([]string{destBasePath}, parts[:i]...)...))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else if err.(*os.PathError).Err == syscall.EINVAL {
				continue
			} else {
				ioError(err)
			}
		}
		panic(BreakoutError.New("placefile: refusing to traverse symlink at %q->%q while placing %q", filepath.Join(parts[:i]...), target, hdr.Name))
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		if hdr.Name == "." {
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
		file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, mode)
		if err != nil {
			ioError(err)
		}
		if _, err := io.Copy(file, body); err != nil {
			file.Close()
			ioError(err)
		}
		file.Close()
	case tar.TypeSymlink:
		// linkname can be anything you want.  it can be invalid, it can be absolute, whatever.
		// the consumer had better know how to jail this filesystem before using;
		// other PlaceFile calls know enough to refuse to traverse this.
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

// Exposed only because you're probably doing your own trees somehow, and it's
// necessary to cover your tracks by forcing times on dirs after all children are done.
// Symmetric params and error handling to `PlaceFile` for your convenience.
func PlaceDirTime(destBasePath string, hdr Metadata) {
	destPath := filepath.Join(destBasePath, hdr.Name)
	if err := fspatch.LUtimesNano(destPath, hdr.AccessTime, hdr.ModTime); err != nil {
		ioError(err)
	}
}

/*
	Scan file attributes into a repeatr Metadata struct, and return an
	`io.Reader` for the file content.

	FileInfo may be provided if it is already available (this will save a stat call).
	The path is expected to exist (nonexistence is a panicable offense, along
	with all other IO errors).

	The reader is nil if the path is any type other than a file.  If a
	reader is returned, the caller is expected to close it.
*/
func ScanFile(basePath, path string, optional ...os.FileInfo) (hdr Metadata, file io.ReadCloser) {
	fullPath := filepath.Join(basePath, path)
	// most of the heavy work is in ReadMetadata; this method just adds the file content
	hdr = ReadMetadata(fullPath, optional...)
	hdr.Name = path
	switch hdr.Typeflag {
	case tar.TypeReg:
		var err error
		file, err = os.OpenFile(fullPath, os.O_RDONLY, 0)
		if err != nil {
			ioError(err)
		}
	}
	return
}
