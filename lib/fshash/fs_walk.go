package fshash

import (
	"hash"
	"io"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/lib/fspatch"
	"polydawn.net/repeatr/lib/treewalk"
)

type fileWalkNode struct {
	path string // relative to start
	info os.FileInfo
	ferr error

	children []*fileWalkNode // note we didn't sort this
	itrIndex int             // next child offset
}

func newFileWalkNode(basePath, path string) (filenode *fileWalkNode) {
	filenode = &fileWalkNode{path: path}
	filenode.info, filenode.ferr = os.Lstat(filepath.Join(basePath, path))
	// don't expand the children until the previsit function
	// we don't want them all crashing into memory at once
	return
}

/*
	Expand next subtree.  Used in the pre-order visit step so we don't walk
	every dir up front.
*/
func (t *fileWalkNode) prepareChildren(basePath string) error {
	if !t.info.IsDir() {
		return nil
	}
	f, err := os.Open(filepath.Join(basePath, t.path))
	if err != nil {
		return err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return err
	}
	t.children = make([]*fileWalkNode, len(names))
	for i, name := range names {
		t.children[i] = newFileWalkNode(basePath, "./"+filepath.Join(t.path, name))
	}
	return nil
}

func (t *fileWalkNode) NextChild() treewalk.Node {
	if t.itrIndex >= len(t.children) {
		return nil
	}
	t.itrIndex++
	return t.children[t.itrIndex-1]
}

func FillBucket(srcBasePath, destBasePath string, bucket Bucket, hasherFactory func() hash.Hash) error {
	// If copying: Dragons: you can set atime and you can set mtime, but you can't ever set ctime again.
	// Filesystem APIs are constructed such that it's literally impossible to do an attribute-preserving copy in userland.

	preVisit := func(node treewalk.Node) error {
		filenode := node.(*fileWalkNode)
		if filenode.ferr != nil {
			return filenode.ferr
		}
		srcPath := filepath.Join(srcBasePath, filenode.path)
		destPath := filepath.Join(destBasePath, filenode.path)
		mode := filenode.info.Mode()
		switch {
		case mode&os.ModeDir == os.ModeDir:
			hdr := ReadMetadata(destPath, filenode.info)
			hdr.Name = filenode.path
			if destBasePath != "" {
				if err := os.MkdirAll(destPath, mode&os.ModePerm); err != nil {
					return err
				}
				// setting time is done in the post-order phase of traversal since adding children will mutate mtime
				if err := os.Chown(destPath, hdr.Uid, hdr.Gid); err != nil {
					return err
				}
			}
			bucket.Record(hdr, nil)
			filenode.prepareChildren(srcBasePath)
		case mode&os.ModeSymlink == os.ModeSymlink:
			var link string
			var err error
			if link, err = os.Readlink(srcPath); err != nil {
				return err
			}
			hdr := ReadMetadata(srcPath, filenode.info)
			hdr.Name = filenode.path
			if destBasePath != "" {
				if err := os.Symlink(link, destPath); err != nil {
					return err
				}
				if err := fspatch.LUtimesNano(destPath, def.Somewhen, hdr.ModTime); err != nil {
					return err
				}
				if err := os.Lchown(destPath, hdr.Uid, hdr.Gid); err != nil {
					return err
				}
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
			hdr := ReadMetadata(destPath, filenode.info)
			hdr.Name = filenode.path
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
	postVisit := func(node treewalk.Node) error {
		filenode := node.(*fileWalkNode)
		filenode.children = nil
		if filenode.info.IsDir() && destBasePath != "" {
			if err := fspatch.UtimesNano(filepath.Join(destBasePath, filenode.path), def.Somewhen, filenode.info.ModTime()); err != nil {
				return err
			}
		}
		return nil
	}

	return treewalk.Walk(newFileWalkNode(srcBasePath, "."), preVisit, postVisit)
}
