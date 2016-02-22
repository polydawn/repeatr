package fs

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
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
	case tar.TypeReg:
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
		panic(errors.ProgrammerError.New("placefile: unhandled file mode %q", hdr.Typeflag))
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

/*
	Alias for `MkdirAllWithAttribs` using sane defaults for metadata (epoch mtime, etc).
*/
func MkdirAll(path string) error {
	return MkdirAllWithAttribs(path, Metadata{
		Mode:       0755,
		ModTime:    def.Epochwhen,
		AccessTime: def.Epochwhen,
		Uid:        0,
		Gid:        0,
	})
}

/*
	Much like `os.MkdirAll`, but standardizes values (mtime, etc) to match the
	given metadata as it goes.

	Existing directories are not modified; the metadata is applied only to
	newly created directories.

	Note that "exising directories are not modified" should be read as "...intentionally".
	As usual, a filesystem with mtime and atime behaviors enabled may change
	those attributes on the top level dir.  If this function created any new dirs,
	it attempt to modify the mtime of the parent to replace its original value; but
	note that this is inherently a best-effort scenario and subject to races.
*/
func MkdirAllWithAttribs(path string, hdr Metadata) error {
	stack, topMTime, err := mkdirAll(path, hdr)
	if err != nil {
		return err
	}
	if stack == nil {
		return nil
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if err := fspatch.LUtimesNano(stack[i], hdr.AccessTime, hdr.ModTime); err != nil {
			return err
		}
	}
	top := filepath.Dir(stack[0])
	if top != "." {
		if err := fspatch.LUtimesNano(top, def.Epochwhen, topMTime); err != nil {
			// gave up and reset atime to epoch.  sue me.  atimes are ridiculous.
			return err
		}
	}
	return nil
}

func mkdirAll(path string, hdr Metadata) (stack []string, topMTime time.Time, err error) {
	// Following code derives from the golang standard library, so you can consider it BSDish if you like.
	// Our changes are licensed under Apache for the sake of overall license simplicity of the project.
	// Ref: https://github.com/golang/go/blob/883bc6ed0ea815293fe6309d66f967ea60630e87/src/os/path.go#L12

	// Fast path: if we can tell whether path is a directory or file, stop with success or error.
	dir, err := os.Stat(path)
	if err == nil {
		if dir.IsDir() {
			return nil, dir.ModTime(), nil
		}
		return nil, dir.ModTime(), &os.PathError{"mkdir", path, syscall.ENOTDIR}
	}

	// Slow path: make sure parent exists and then call Mkdir for path.
	i := len(path)
	for i > 0 && os.IsPathSeparator(path[i-1]) { // Skip trailing path separator.
		i--
	}

	j := i
	for j > 0 && !os.IsPathSeparator(path[j-1]) { // Scan backward over element.
		j--
	}

	if j > 1 {
		// Create parent
		stack, topMTime, err = mkdirAll(path[0:j-1], hdr)
		if err != nil {
			return stack, topMTime, err
		}
	}

	// Parent now exists; invoke Mkdir and use its result.
	err = os.Mkdir(path, 0755)
	if err != nil {
		// Handle arguments like "foo/." by
		// double-checking that directory doesn't exist.
		dir, err1 := os.Lstat(path)
		if err1 == nil && dir.IsDir() {
			return stack, topMTime, nil
		}
		return stack, topMTime, err
	}
	stack = append(stack, path)

	// Apply standardizations.
	if err := os.Lchown(path, hdr.Uid, hdr.Gid); err != nil {
		return stack, topMTime, err
	}
	if err := os.Chmod(path, hdr.FileMode()); err != nil {
		return stack, topMTime, err
	}
	// Except for time, because as usual with dirs, that requires walking backwards again at the end.
	// That'll be done one function out.

	return stack, topMTime, nil
}

/*
	`os.Chown`, recursively.
*/
func Chownr(path string, uid, gid int) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := os.Chown(path, uid, gid); err != nil {
			return err
		}
		return nil
	})
}
